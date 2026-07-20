// Liner color palette — exactly five colors, no more.
//
// We use ONLY these hex values across the TUI. If you find yourself reaching
// for "cyan", "red", "white", "black", or any other named color, stop and
// pick one of the five below. The constraint is intentional — five colors,
// consistently applied, gives the product a recognizable signature and
// keeps the visual language tight.
//
// Palette
// =======
//
//   #ff4500  red-orange     — main brand color: structure, interactivity
//   #00ac11  green          — success / completion / progress
//   #b68800  goldenrod      — attention, "look here now" / AI running
//   #ab65ff  purple         — heading / brand accent
//   #ff0086  hot pink       — alarm / error
//   #daf1fd  pale blue      — light-on-dark text (chip labels, highlights)
//
// Roles
// =====
//
// STRUCTURE  (red-orange)   — borders, chip backgrounds, hotkey labels, "this
//                              is interactive" affordances, selection
//                              highlight backgrounds. The main brand color and
//                              the structural skeleton of every screen. Text
//                              placed ON a STRUCTURE background must be light
//                              (white / HIGHLIGHT), never black.
//
// CURRENT    (yellow)       — "you are here." The currently-selected menu
//                              item, the chip on the focus card when its
//                              phase is in_progress, the active selection in
//                              every list. NEVER use for borders.
//
// HEADING    (purple)       — content section headers. Markdown `**Heading:**`
//                              runs, wizard step questions, the "Get started"
//                              label on the splash, "Review:" before the
//                              confirmation table.
//
// SUCCESS    (green)        — "done." ✓ glyphs, completed phases, accepted
//                              gates, a fully-succeeded compile, the progress
//                              indicator's filled portion. A distinct green,
//                              no longer the same hue as STRUCTURE.
//
// WARNING    (yellow)       — partial / draft-pending / "in flight, look at
//                              this." Doubles up with CURRENT — both are
//                              attention signals; the only difference is
//                              whether the user caused it (CURRENT) or the
//                              system did (WARNING).
//
// ERROR      (magenta-pink) — "failed." ✗ glyphs, failed compiles, error
//                              messages. Loud and unambiguous against the
//                              other four colors.
//
// HIGHLIGHT  (pale blue)    — light text on dark chip backgrounds (replaces
//                              the legacy "black on cyan" pattern), and any
//                              place a foreground text accent should pop
//                              against the default body color.
//
// MUTED      (grey #9a9a9a)   — secondary content. Replaces Ink's `dimColor`
//                              prop, which dims relative to the terminal's
//                              foreground and renders near-invisible on many
//                              themes. An explicit grey guarantees legible
//                              contrast on dark backgrounds while still
//                              reading as clearly secondary.
//
// Default    (no color)     — primary body text. Most content lands here so
//                              the colored accents have somewhere to pop.
//
// Box color rule (the only rule)
// ==============================
//
// **Border and chip share the same color**, always. The chip is a solid
// section of the top border line; visually they're one element, so they
// can never disagree.
//
//   default state             → STRUCTURE   (teal)
//   current / "you are here"  → CURRENT     (yellow)
//   running / in-flight       → CURRENT     (yellow) — same as current
//   done / accepted           → SUCCESS     (teal)
//   partial / warn / draft    → WARNING     (yellow)
//   failed                    → ERROR       (magenta-pink)
//
// Pass exactly ONE color to `LabeledBox` via the `color` prop and the chip
// background follows automatically. Don't override `labelBg` separately
// unless you have a specific reason (e.g. the Header brand chip).

// Raw palette values. Prefer the role aliases below — these are exported
// for the cassette logo and any other place that wants the full palette
// (e.g. a random pick per character).
// Constant names (TEAL / YELLOW / MAGENTA) describe the role they
// historically held, not the literal hex. Callers import the role aliases
// below (STRUCTURE / CURRENT / HEADING / etc.) — renaming the raw
// constants would churn unrelated files without changing behavior.
export const COLOR_ORANGE = "#ff4500"; // red-orange — the main brand color
export const COLOR_TEAL = "#00ac11"; // a true green (success / progress)
export const COLOR_YELLOW = "#b68800"; // a goldenrod (attention / running)
export const COLOR_PURPLE = "#ab65ff"; // a vibrant purple (headings)
export const COLOR_MAGENTA = "#ff0086"; // a hot pink (errors)
export const COLOR_PALE = "#daf1fd";
export const COLOR_GREY = "#9a9a9a";
export const COLOR_BLACK = "#000000";

/** The full palette as an array — useful for randomized rendering. */
export const PALETTE: readonly string[] = [
  COLOR_ORANGE,
  COLOR_TEAL,
  COLOR_YELLOW,
  COLOR_PURPLE,
  COLOR_MAGENTA,
  COLOR_PALE,
];

// Role aliases — what every screen should import.
//
// STRUCTURE and SUCCESS are intentionally DIFFERENT colors now: STRUCTURE is
// the red-orange brand color used for borders, chip backgrounds, selection
// highlights, and hotkey labels; SUCCESS is green and reserved for ✓ glyphs,
// completed phases, and the progress indicator. Anything selected with a
// STRUCTURE background should use white/HIGHLIGHT text, not black — the
// orange background needs a light foreground for contrast.
export const STRUCTURE = COLOR_ORANGE;
export const CURRENT = COLOR_YELLOW;
export const HEADING = COLOR_PURPLE;
export const SUCCESS = COLOR_TEAL;
export const WARNING = COLOR_YELLOW;
export const ERROR = COLOR_MAGENTA;
export const HIGHLIGHT = COLOR_PALE;
export const MUTED = COLOR_GREY;
// ON_FILL — true-black foreground for text sitting on a solid colored
// background (the STRUCTURE/orange selection highlight, LabeledBox label
// chips, etc.). Use this instead of the named color "black": ANSI "black"
// (color 0) renders as dark grey on many terminal themes and fails contrast
// against the orange fill. An explicit hex guarantees true black everywhere.
export const ON_FILL = COLOR_BLACK;
