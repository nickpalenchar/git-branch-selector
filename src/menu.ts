import { terminal } from 'terminal-kit';

// Properly handle Ctrl+C (SIGINT)
function exitGracefully() {
    process.exit(0);
}

// Capture SIGINT from process
process.on('SIGINT', exitGracefully);

async function runMenu() {
    const originalOptions = process.argv.slice(2);

    if (!originalOptions.length) {
        terminal.red("\nNo options provided. Usage:\n");
        terminal.green("   bun run menu.ts option1 option2 option3\n\n");
        process.exit(1);
    }

    terminal.clear();

    const menuWidth = Math.max(Math.max(...originalOptions.map(opt => opt.length)) + 14, 60);

    let filterText = "";
    let filteredOptions = [...originalOptions];
    let selectedIndex = 0;

    function renderMenu() {
        terminal.clear();

        terminal.moveTo(1, 1);
        terminal.brightCyan("╭" + "─".repeat(menuWidth) + "╮\n");
        terminal.brightCyan("│ ").brightWhite.bold("Select an option:").column(menuWidth + 2)("│\n");
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
        } else if (char.length === 1 && char.match(/[a-zA-Z0-9 ]/)) {
            filterText += char;
        }

        filteredOptions = originalOptions.filter(opt => opt.toLowerCase().includes(filterText.toLowerCase()));
        selectedIndex = 0;
        renderMenu();
    }

    function handleKey(key: string) {
        if (key === "UP" && filteredOptions.length > 0) {
            selectedIndex = (selectedIndex - 1 + filteredOptions.length) % filteredOptions.length;
        } else if (key === "DOWN" && filteredOptions.length > 0) {
            selectedIndex = (selectedIndex + 1) % filteredOptions.length;
        } else if (key === "ENTER") {
            if (filteredOptions[selectedIndex] === undefined) {
                return;
            }
            terminal("\nYou selected: ").green.bold(filteredOptions[selectedIndex] + "\n\n");
            process.exit(0);
        } else if (key === "CTRL_C") {
            exitGracefully();
        } else {
            updateFilter(key);
        }
        renderMenu();
    }

    renderMenu();

    // Enable live key handling and ensure Ctrl+C works
    terminal.grabInput({ mouse: "button" });
    terminal.on("key", handleKey);
}

runMenu();
