# Maze Forge

Static frontend for generating 2D mazes directly in the browser.

## What it does

- Generates a segmented 2D maze on the client side.
- Uses a seed, so the same settings reproduce the same maze.
- Shows four exits where only one is reachable from the entrance.
- Exports the rendered maze as a PNG.
- Stores settings in the URL hash and local storage.

## Serverless generation

Maze generation happens entirely in `app.js`. There are no network calls and no backend dependency for the generator.

## Run

Open `index.html` in a browser, or serve the repository with any static file server.

Examples:

```bash
python -m http.server 8080
```

```bash
npx serve .
```
