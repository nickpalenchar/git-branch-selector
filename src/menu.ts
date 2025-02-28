import { terminal } from "terminal-kit";
import { execSync } from "child_process";

process.on("SIGINT", () => {
    terminal("\n");
    process.exit(0);
});

let handleKey: (key: string) => void;
let branchSelected = false;

function getGitBranches(): string[] {
    try {
        const currentBranch = execSync("git branch --show-current", { encoding: "utf-8", stdio: "pipe" }).trim();
        const reflogBranches = execSync(
            "git reflog show --pretty=format:'%gs' | grep 'checkout:' | grep -oE '[^ ]+$' | awk '!seen[$0]++' | head -n 17",
            { encoding: "utf-8", stdio: "pipe" }
        ).trim().split("\n").filter((b) => b !== currentBranch);

        if (reflogBranches.length > 0 && reflogBranches[0] !== "") return reflogBranches;

        return execSync("git branch --format='%(refname:short)'", { encoding: "utf-8", stdio: "pipe" })
            .trim().split("\n").filter((b) => b !== currentBranch);
    } catch (error) {
        return [];
    }
}

function isWorkingDirectoryDirty(): boolean {
    try {
        const status = execSync("git status --porcelain=v1 | grep '^ M'", { encoding: "utf-8", stdio: "pipe" });
        return status.trim().length > 0;
    } catch (error) {
        return false;
    }
}

async function confirmAndStash(): Promise<boolean> {
    terminal.brightRed("\nYour working directory has uncommitted changes.\n");
    terminal.brightYellow("Stash changes before switching? (Y/n): ");

    return new Promise((resolve) => {
        terminal.inputField({ default: "" }, (error, input) => {
            if (error || input === undefined) {
                terminal("\n");
                process.exit(0);
            }
            if (input.toLowerCase() !== "n") {
                try {
                    terminal.brightBlue("\nStashing changes...\n");
                    execSync("git stash", { stdio: "inherit" });
                    resolve(true);
                } catch (stashError) {
                    terminal.red("\nFailed to stash changes. Aborting branch switch.\n");
                    process.exit(1);
                }
            } else {
                resolve(false);
            }
        });
    });
}

async function checkoutBranch(branch: string) {
    terminal.grabInput(false);
    terminal.off("key", handleKey);

    terminal.clear();
    
    if (isWorkingDirectoryDirty()) {
        const shouldStash = await confirmAndStash();
        if (!shouldStash) {
            terminal.red("\nBranch switch canceled due to uncommitted changes.\n");
            process.exit(0);
        }
    }
    
    terminal.brightGreen(`\nSwitching to branch: ${branch}...\n`);
    try {
        execSync(`git checkout ${branch}`, { stdio: "inherit" });
        process.exit(0);
    } catch (error) {
        terminal.red(`\nFailed to checkout branch: ${branch}\n`);
        process.exit(1);
    }
}

async function runMenu() {
    const branches = getGitBranches();
    if (branches.length === 0) {
        terminal.red("\nNo branches found.\n");
        process.exit(1);
    }

    terminal.clear();
    const menuWidth = Math.max(Math.max(...branches.map((opt) => opt.length)) + 14, 60);
    let filterText = "";
    let filteredOptions = [...branches];
    let selectedIndex = 0;

    function renderMenu() {
        if (branchSelected) {
            return;
        }
        terminal.clear();
        terminal.moveTo(1, 1);
        terminal.brightCyan("╭" + "─".repeat(menuWidth) + "╮\n");
        terminal.brightCyan("│ ").brightWhite.bold("Select a branch:").column(menuWidth + 2)("│\n");
        terminal.brightCyan("│ ").gray("Filter: ").brightWhite(filterText.padEnd(menuWidth - 9)).column(menuWidth + 2)("│\n");
        terminal.brightCyan("├" + "─".repeat(menuWidth) + "┤\n");

        if (filteredOptions.length === 0) {
            terminal.brightRed("│ No matches found.").column(menuWidth + 2)("│\n");
        } else {
            filteredOptions.forEach((option, i) => {
                terminal.brightCyan("│ ");
                if (i === selectedIndex) {
                    terminal.brightGreen.bold("> " + option.padEnd(menuWidth - 4)).column(menuWidth + 2)("│\n");
                } else {
                    terminal("  " + option.padEnd(menuWidth - 4)).column(menuWidth + 2)("│\n");
                }
            });
        }

        terminal.brightCyan("╰" + "─".repeat(menuWidth) + "╯\n");
    }

    function updateFilter(char: string) {
        if (char === "BACKSPACE") {
            filterText = filterText.slice(0, -1);
        } else if (char.length === 1 && char.match(/[a-zA-Z0-9-_]/)) {
            filterText += char;
        }

        filteredOptions = branches.filter((opt) =>
            opt.toLowerCase().includes(filterText.toLowerCase())
        );
        selectedIndex = 0;
        renderMenu();
    }

    handleKey = function (key: string) {
        if (key === "CTRL_C") {
            terminal("\n");
            process.exit(0);
        }

        if (key === "UP" && filteredOptions.length > 0) {
            selectedIndex = (selectedIndex - 1 + filteredOptions.length) % filteredOptions.length;
        } else if (key === "DOWN" && filteredOptions.length > 0) {
            selectedIndex = (selectedIndex + 1) % filteredOptions.length;
        } else if (key === "ENTER") {
            if (filteredOptions[selectedIndex] === undefined) {
                return;
            }
            branchSelected = true;
            checkoutBranch(filteredOptions[selectedIndex]);
        } else {
            updateFilter(key);
        }
        renderMenu();
    };

    renderMenu();
    terminal.grabInput({ mouse: "button" });
    terminal.on("key", handleKey);
}

runMenu();
