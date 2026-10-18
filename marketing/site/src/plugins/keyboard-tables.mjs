/** Keep wide reference tables reachable with arrow keys at narrow widths. */
export default {
  name: "keyboard-tables",
  element: {
    filter: ["table"],
    visit(node, context) {
      context.setProperty(node, "tabIndex", 0);
    },
  },
};
