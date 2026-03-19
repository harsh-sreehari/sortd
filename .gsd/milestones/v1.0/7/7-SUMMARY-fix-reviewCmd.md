# Fix Summary: Interactive Review Command

- Altered `reviewCmd` in `main.go`.
- Implemented `bufio.NewScanner(os.Stdin)` iteration checking `.unsorted/` files directly and halting process awaiting STDIN routing via keyboard logic.
- Routed user CLI strings straight to `pipe.Mover.Move()`.
