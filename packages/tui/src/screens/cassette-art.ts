// ASCII art for the boot splash. Each entry in *_ROWS is one row of glyphs;
// the splash prints them as foreground text in the brand color. Rows are
// kept padded to a uniform width so the logo block aligns cleanly in the
// layout regardless of trailing-space stripping.
//
// Two variants ship: the wide CASSETTE_ROWS (100×29) for full-size terminals,
// and a compact SMALL_CASSETTE_ROWS (66×19) for narrower windows that can't
// fit the full art. BootSplash picks the right tier by terminal columns:
//
//   cols >= CASSETTE_WIDTH        → CASSETTE_ROWS (full)
//   cols >= SMALL_CASSETTE_WIDTH  → SMALL_CASSETTE_ROWS (compact)
//   otherwise                      → ink-big-text wordmark fallback
//
// To add another variant: add new *_ROWS + WIDTH/HEIGHT constants and slot
// them into the tier ladder in BootSplash.

export const CASSETTE_WIDTH = 100;
export const CASSETTE_HEIGHT = 29;

export const CASSETTE_ROWS: readonly string[] = [
  "                                                                                                    ",
  "                                                                                                    ",
  "                                                                                                    ",
  "                                                                                                    ",
  "                            ##                                                                      ",
  "                           ###                                                                      ",
  "                          ###                                                                       ",
  "                         ###      ###                                                               ",
  "                        ###      ###                                                                ",
  "                       ###      ###                                                                 ",
  "                      ###       ##                                                  #               ",
  "                     ###                           ####              #####      #######             ",
  "                    ###      ###        ###     ########          #########    #########            ",
  "                    ##      ###         ##    ####   ###        ####  ####       ###########        ",
  "                   ##       ##         ##   ###     ##       ####   ####       ###       ###      ##",
  "                  ###      ###      ##### ###     ###      ##########       ####        ###    #####",
  "                 ###      ###    ##########      ###    #########       ######         ###   ####   ",
  "                ###      ###  ####  #####       ###  ########        #####            ### ####      ",
  "########       ###      ########   ####        #######    ###  #######                ######        ",
  "###   ###########        ###      ###           ###        ######                     ###           ",
  " ###         ###########           #                                                                ",
  "   ###      ###      ###########                                                                    ",
  "     ###   ###                ###########                                                           ",
  "      #######                          ############                                                 ",
  "         ###                                   ################                                     ",
  "                                                          ###########                               ",
  "                                                                                                    ",
  "                                                                                                    ",
  "                                                                                                    ",
];

export const SMALL_CASSETTE_WIDTH = 66;
export const SMALL_CASSETTE_HEIGHT = 19;

export const SMALL_CASSETTE_ROWS: readonly string[] = [
  "                                                                  ",
  "                                                                  ",
  "                                                                  ",
  "                  ##                                              ",
  "                 ##                                               ",
  "                ##    ##                                          ",
  "               ##    ##                                           ",
  "              ##                  ###         ###     ###         ",
  "             ##    ##     ##   ##  ##      ##  ##    #######      ",
  "            ##    ##     ##  ##   ##     ##   ##    ##     ##   # ",
  "           ###   ##    # ####    ##    ######    ###     ##   ##  ",
  "          ###   ##   #  ###     ##    ##      ###        #####    ",
  "###########     #####  ##      ####### ######           ###       ",
  " ###     #########                                                ",
  "  ###  ##         ########                                        ",
  "     ####                  #########                              ",
  "                                    #########                     ",
  "                                                                  ",
  "                                                                  ",
];
