(function () {
  "use strict";

  const EXIT_COUNT = 4;
  const REVEAL_DURATION = 1200;
  const REGENERATE_DELAY = 1100;
  const defaults = {
    width: 24,
    height: 18,
    cellSize: 24,
    seed: "",
    showGrid: false
  };

  const wallMap = {
    north: { dx: 0, dy: -1, opposite: "south" },
    south: { dx: 0, dy: 1, opposite: "north" },
    west: { dx: -1, dy: 0, opposite: "east" },
    east: { dx: 1, dy: 0, opposite: "west" }
  };

  const wallOrder = ["north", "east", "south", "west"];
  const answerLabels = ["A", "B", "C", "D"];

  const form = document.getElementById("controls");
  const frame = document.getElementById("canvas-frame");
  const canvas = document.getElementById("maze-canvas");
  const context = canvas.getContext("2d");
  const answerButtonsNode = document.getElementById("answer-buttons");
  const drawer = document.getElementById("settings-drawer");
  const drawerBackdrop = document.getElementById("drawer-backdrop");

  const metricSeed = document.getElementById("metric-seed");
  const metricRoute = document.getElementById("metric-route");
  const metricDeadEnds = document.getElementById("metric-dead-ends");
  const metricEntrance = document.getElementById("metric-entrance");
  const statusNode = document.getElementById("status");
  const mazeSizeNode = document.getElementById("maze-size");

  const controls = {
    width: document.getElementById("maze-width"),
    height: document.getElementById("maze-height"),
    cellSize: document.getElementById("cell-size"),
    seed: document.getElementById("seed"),
    showGrid: document.getElementById("show-grid"),
    shuffleSeed: document.getElementById("shuffle-seed"),
    copyLink: document.getElementById("copy-link"),
    downloadPng: document.getElementById("download-png"),
    openSettings: document.getElementById("open-settings"),
    closeSettings: document.getElementById("close-settings")
  };

  let currentRender = null;
  let animationFrameId = 0;
  let generationTimerId = 0;
  let regenerateTimerId = 0;
  let isResolvingAnswer = false;

  boot();

  function boot() {
    const initialSettings = readInitialSettings();
    applySettingsToForm(initialSettings);
    bindEvents();
    generateMazeFromForm({ announce: "Лабиринт готов. Выбери один из 4 выходов." });
  }

  function bindEvents() {
    form.addEventListener("submit", function (event) {
      event.preventDefault();
      closeDrawer();
      generateMazeFromForm({ announce: "Параметры обновлены. Выбери выход." });
    });

    controls.shuffleSeed.addEventListener("click", function () {
      controls.seed.value = createSeed();
      closeDrawer();
      generateMazeFromForm({ announce: "Новый seed применён." });
    });

    controls.copyLink.addEventListener("click", copyShareLink);
    controls.downloadPng.addEventListener("click", downloadPng);

    controls.showGrid.addEventListener("change", function () {
      if (!currentRender) {
        return;
      }

      currentRender.settings.showGrid = controls.showGrid.checked;
      persistCurrentSettings();
      drawMaze(currentRender, 0, "idle");
    });

    controls.openSettings.addEventListener("click", openDrawer);
    controls.closeSettings.addEventListener("click", closeDrawer);
    drawerBackdrop.addEventListener("click", closeDrawer);

    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && drawer.classList.contains("is-open")) {
        closeDrawer();
      }
    });

    window.addEventListener("resize", function () {
      if (currentRender) {
        drawMaze(currentRender, currentRender.revealProgress || 0, currentRender.feedbackState || "idle");
      }
    });
  }

  function openDrawer() {
    drawer.hidden = false;
    drawer.classList.add("is-open");
    drawer.setAttribute("aria-hidden", "false");
    drawerBackdrop.hidden = false;
    requestAnimationFrame(function () {
      drawerBackdrop.classList.add("is-visible");
    });
    document.body.classList.add("drawer-open");
    controls.openSettings.setAttribute("aria-expanded", "true");
  }

  function closeDrawer() {
    drawer.classList.remove("is-open");
    drawer.setAttribute("aria-hidden", "true");
    drawerBackdrop.classList.remove("is-visible");
    document.body.classList.remove("drawer-open");
    controls.openSettings.setAttribute("aria-expanded", "false");
    window.setTimeout(function () {
      if (!drawer.classList.contains("is-open")) {
        drawer.hidden = true;
        drawerBackdrop.hidden = true;
      }
    }, 240);
  }

  function readInitialSettings() {
    const fromHash = parseHash(window.location.hash);
    const fromStorage = readStorage();
    const merged = Object.assign({}, defaults, fromStorage, fromHash);

    if (!merged.seed) {
      merged.seed = createSeed();
    }

    return normalizeSettings(merged);
  }

  function parseHash(hashValue) {
    if (!hashValue || hashValue.length < 2) {
      return {};
    }

    const query = new URLSearchParams(hashValue.replace(/^#/, ""));

    return {
      width: query.get("w"),
      height: query.get("h"),
      cellSize: query.get("c"),
      seed: query.get("s") || "",
      showGrid: parseBoolean(query.get("grid"), defaults.showGrid)
    };
  }

  function parseBoolean(value, fallback) {
    if (value === "1" || value === "true") {
      return true;
    }

    if (value === "0" || value === "false") {
      return false;
    }

    return fallback;
  }

  function readStorage() {
    try {
      const raw = window.localStorage.getItem("maze-forge-settings");
      return raw ? JSON.parse(raw) : {};
    } catch (error) {
      return {};
    }
  }

  function writeStorage(settings) {
    try {
      window.localStorage.setItem("maze-forge-settings", JSON.stringify(settings));
    } catch (error) {
      return;
    }
  }

  function applySettingsToForm(settings) {
    controls.width.value = settings.width;
    controls.height.value = settings.height;
    controls.cellSize.value = settings.cellSize;
    controls.seed.value = settings.seed;
    controls.showGrid.checked = settings.showGrid;
  }

  function normalizeSettings(source) {
    return {
      width: clampNumber(source.width, 5, 80, defaults.width),
      height: clampNumber(source.height, 5, 80, defaults.height),
      cellSize: clampNumber(source.cellSize, 12, 40, defaults.cellSize),
      seed: String(source.seed || "").trim(),
      showGrid: Boolean(source.showGrid)
    };
  }

  function clampNumber(value, min, max, fallback) {
    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed)) {
      return fallback;
    }

    return Math.min(max, Math.max(min, parsed));
  }

  function generateMazeFromForm(options) {
    const settings = normalizeSettings({
      width: controls.width.value,
      height: controls.height.value,
      cellSize: controls.cellSize.value,
      seed: controls.seed.value,
      showGrid: controls.showGrid.checked
    });

    if (!settings.seed) {
      settings.seed = createSeed();
      controls.seed.value = settings.seed;
    }

    frame.classList.remove("is-success", "is-error");
    frame.classList.add("is-refreshing");
    isResolvingAnswer = false;

    cancelAnimationFrame(animationFrameId);
    window.clearTimeout(generationTimerId);
    window.clearTimeout(regenerateTimerId);

    generationTimerId = window.setTimeout(function () {
      const renderModel = buildRenderModel(settings);
      currentRender = renderModel;
      drawMaze(renderModel, 0, "idle");
      renderAnswerButtons(renderModel);
      updateSummary(renderModel);
      writeHash(settings);
      writeStorage(settings);
      frame.classList.remove("is-refreshing");

      if (options && options.announce) {
        setStatus(options.announce);
      }
    }, 90);
  }

  function buildRenderModel(settings) {
    const rng = createRng(settings.seed);
    const grid = createGrid(settings.width, settings.height);
    carveMaze(grid, settings.width, settings.height, rng);

    const components = splitMazeIntoComponents(grid, settings.width, settings.height, rng);
    const entranceRegion = components[Math.floor(rng() * components.length)];
    const entrance = createEntrance(grid, settings.width, settings.height, rng, entranceRegion);
    const exitLayout = createExits(grid, settings.width, settings.height, rng, entrance, components, entranceRegion);
    const exits = exitLayout.exits;
    const correctExit = exitLayout.correctExit;
    const route = findRoute(grid, settings.width, settings.height, entrance, correctExit);
    const deadEnds = countDeadEnds(grid);

    return {
      settings: Object.assign({}, settings),
      grid: grid,
      entrance: entrance,
      exits: exits,
      correctExit: correctExit,
      route: route,
      deadEnds: deadEnds,
      pixelWidth: settings.width * settings.cellSize,
      pixelHeight: settings.height * settings.cellSize,
      revealProgress: 0,
      feedbackState: "idle"
    };
  }

  function createGrid(width, height) {
    const grid = [];

    for (let y = 0; y < height; y += 1) {
      const row = [];
      for (let x = 0; x < width; x += 1) {
        row.push({
          x: x,
          y: y,
          visited: false,
          walls: {
            north: true,
            east: true,
            south: true,
            west: true
          }
        });
      }
      grid.push(row);
    }

    return grid;
  }

  function carveMaze(grid, width, height, rng) {
    const startX = Math.floor(rng() * width);
    const startY = Math.floor(rng() * height);
    const stack = [grid[startY][startX]];
    grid[startY][startX].visited = true;

    while (stack.length > 0) {
      const current = stack[stack.length - 1];
      const neighbors = [];

      for (let i = 0; i < wallOrder.length; i += 1) {
        const direction = wallOrder[i];
        const rule = wallMap[direction];
        const nextX = current.x + rule.dx;
        const nextY = current.y + rule.dy;

        if (nextX < 0 || nextX >= width || nextY < 0 || nextY >= height) {
          continue;
        }

        const nextCell = grid[nextY][nextX];
        if (!nextCell.visited) {
          neighbors.push({ direction: direction, cell: nextCell });
        }
      }

      if (neighbors.length === 0) {
        stack.pop();
        continue;
      }

      const choice = neighbors[Math.floor(rng() * neighbors.length)];
      const currentWall = choice.direction;
      const nextWall = wallMap[currentWall].opposite;

      current.walls[currentWall] = false;
      choice.cell.walls[nextWall] = false;
      choice.cell.visited = true;
      stack.push(choice.cell);
    }
  }

  function createEntrance(grid, width, height, rng, region) {
    const candidates = borderCandidatesForRegion(width, height, region);
    const choice = candidates[Math.floor(rng() * candidates.length)];
    grid[choice.y][choice.x].walls[choice.side] = false;

    return {
      x: choice.x,
      y: choice.y,
      side: choice.side,
      label: "IN"
    };
  }

  function createExits(grid, width, height, rng, entrance, components, entranceRegion) {
    const exits = [];
    const correctExit = createExitInRegion(grid, width, height, rng, entranceRegion, entrance);
    correctExit.isCorrect = true;
    exits.push(correctExit);

    for (let index = 0; index < components.length; index += 1) {
      const region = components[index];
      if (region === entranceRegion) {
        continue;
      }

      exits.push(createExitInRegion(grid, width, height, rng, region, null));
    }

    shuffle(exits, rng);
    exits.forEach(function (exit, index) {
      exit.label = answerLabels[index];
    });

    return {
      exits: exits,
      correctExit: exits.find(function (exit) {
        return exit.isCorrect;
      })
    };
  }

  function createExitInRegion(grid, width, height, rng, region, excludedPoint) {
    const candidates = borderCandidatesForRegion(width, height, region).filter(function (candidate) {
      if (!excludedPoint) {
        return true;
      }

      return !(candidate.x === excludedPoint.x && candidate.y === excludedPoint.y);
    });
    const choice = candidates[Math.floor(rng() * candidates.length)];
    grid[choice.y][choice.x].walls[choice.side] = false;

    return {
      x: choice.x,
      y: choice.y,
      side: choice.side,
      label: ""
    };
  }

  function borderCandidatesForRegion(width, height, region) {
    return region.borderCandidates.slice();
  }

  function splitMazeIntoComponents(grid, width, height, rng) {
    const attempts = 240;
    const minComponentSize = Math.max(6, Math.floor((width * height) / 12));
    const edges = listOpenEdges(grid, width, height);

    for (let attempt = 0; attempt < attempts; attempt += 1) {
      const blockedEdges = pickBlockedEdges(edges, rng);
      const components = findComponents(grid, width, height, blockedEdges);
      const valid = components.length === EXIT_COUNT && components.every(function (component) {
        return component.cells.length >= minComponentSize && component.borderCandidates.length > 0;
      });

      if (!valid) {
        continue;
      }

      applyBlockedEdges(grid, blockedEdges);
      return components;
    }

    throw new Error("Failed to split maze into four irregular components.");
  }

  function listOpenEdges(grid, width, height) {
    const edges = [];

    for (let y = 0; y < height; y += 1) {
      for (let x = 0; x < width; x += 1) {
        const cell = grid[y][x];

        if (!cell.walls.east && x + 1 < width) {
          edges.push({
            fromX: x,
            fromY: y,
            toX: x + 1,
            toY: y,
            direction: "east"
          });
        }

        if (!cell.walls.south && y + 1 < height) {
          edges.push({
            fromX: x,
            fromY: y,
            toX: x,
            toY: y + 1,
            direction: "south"
          });
        }
      }
    }

    return edges;
  }

  function pickBlockedEdges(edges, rng) {
    const pool = edges.slice();
    const blocked = new Set();

    shuffle(pool, rng);

    for (let index = 0; index < EXIT_COUNT - 1; index += 1) {
      blocked.add(edgeKey(pool[index].fromX, pool[index].fromY, pool[index].toX, pool[index].toY));
    }

    return blocked;
  }

  function findComponents(grid, width, height, blockedEdges) {
    const visited = new Set();
    const components = [];

    for (let y = 0; y < height; y += 1) {
      for (let x = 0; x < width; x += 1) {
        const startKey = pointKey(x, y);
        if (visited.has(startKey)) {
          continue;
        }

        const queue = [{ x: x, y: y }];
        const cells = [];
        const cellKeys = new Set([startKey]);
        visited.add(startKey);

        while (queue.length > 0) {
          const current = queue.shift();
          const cell = grid[current.y][current.x];
          cells.push(current);

          for (let index = 0; index < wallOrder.length; index += 1) {
            const direction = wallOrder[index];
            const rule = wallMap[direction];

            if (cell.walls[direction]) {
              continue;
            }

            const nextX = current.x + rule.dx;
            const nextY = current.y + rule.dy;

            if (nextX < 0 || nextX >= width || nextY < 0 || nextY >= height) {
              continue;
            }

            if (blockedEdges.has(edgeKey(current.x, current.y, nextX, nextY))) {
              continue;
            }

            const nextKey = pointKey(nextX, nextY);
            if (visited.has(nextKey)) {
              continue;
            }

            visited.add(nextKey);
            cellKeys.add(nextKey);
            queue.push({ x: nextX, y: nextY });
          }
        }

        components.push({
          cells: cells,
          borderCandidates: buildBorderCandidatesForCells(cells, width, height)
        });
      }
    }

    return components;
  }

  function buildBorderCandidatesForCells(cells, width, height) {
    const candidates = [];

    for (let index = 0; index < cells.length; index += 1) {
      const cell = cells[index];

      if (cell.y === 0) {
        candidates.push({ x: cell.x, y: cell.y, side: "north" });
      }
      if (cell.y === height - 1) {
        candidates.push({ x: cell.x, y: cell.y, side: "south" });
      }
      if (cell.x === 0) {
        candidates.push({ x: cell.x, y: cell.y, side: "west" });
      }
      if (cell.x === width - 1) {
        candidates.push({ x: cell.x, y: cell.y, side: "east" });
      }
    }

    return candidates;
  }

  function applyBlockedEdges(grid, blockedEdges) {
    blockedEdges.forEach(function (key) {
      const edge = parseEdgeKey(key);
      const fromCell = grid[edge.fromY][edge.fromX];
      const toCell = grid[edge.toY][edge.toX];

      if (edge.fromX === edge.toX) {
        if (edge.fromY < edge.toY) {
          fromCell.walls.south = true;
          toCell.walls.north = true;
        } else {
          fromCell.walls.north = true;
          toCell.walls.south = true;
        }
        return;
      }

      if (edge.fromX < edge.toX) {
        fromCell.walls.east = true;
        toCell.walls.west = true;
      } else {
        fromCell.walls.west = true;
        toCell.walls.east = true;
      }
    });
  }

  function edgeKey(x1, y1, x2, y2) {
    if (x1 < x2 || (x1 === x2 && y1 <= y2)) {
      return String(x1) + ":" + String(y1) + "|" + String(x2) + ":" + String(y2);
    }

    return String(x2) + ":" + String(y2) + "|" + String(x1) + ":" + String(y1);
  }

  function parseEdgeKey(key) {
    const parts = key.split("|");
    const start = parts[0].split(":");
    const end = parts[1].split(":");

    return {
      fromX: Number.parseInt(start[0], 10),
      fromY: Number.parseInt(start[1], 10),
      toX: Number.parseInt(end[0], 10),
      toY: Number.parseInt(end[1], 10)
    };
  }

  function borderCandidates(width, height) {
    const candidates = [];

    for (let x = 0; x < width; x += 1) {
      candidates.push({ x: x, y: 0, side: "north" });
      if (height > 1) {
        candidates.push({ x: x, y: height - 1, side: "south" });
      }
    }

    for (let y = 1; y < height - 1; y += 1) {
      candidates.push({ x: 0, y: y, side: "west" });
      if (width > 1) {
        candidates.push({ x: width - 1, y: y, side: "east" });
      }
    }

    return candidates;
  }

  function shuffle(items, rng) {
    for (let index = items.length - 1; index > 0; index -= 1) {
      const swapIndex = Math.floor(rng() * (index + 1));
      const current = items[index];
      items[index] = items[swapIndex];
      items[swapIndex] = current;
    }
  }

  function findRoute(grid, width, height, startPoint, endPoint) {
    const queue = [{ x: startPoint.x, y: startPoint.y }];
    const visited = new Set([pointKey(startPoint.x, startPoint.y)]);
    const previous = new Map();

    while (queue.length > 0) {
      const current = queue.shift();
      if (current.x === endPoint.x && current.y === endPoint.y) {
        break;
      }

      const cell = grid[current.y][current.x];

      for (let i = 0; i < wallOrder.length; i += 1) {
        const direction = wallOrder[i];
        const rule = wallMap[direction];

        if (cell.walls[direction]) {
          continue;
        }

        const nextX = current.x + rule.dx;
        const nextY = current.y + rule.dy;

        if (nextX < 0 || nextX >= width || nextY < 0 || nextY >= height) {
          continue;
        }

        const key = pointKey(nextX, nextY);
        if (visited.has(key)) {
          continue;
        }

        visited.add(key);
        previous.set(key, current);
        queue.push({ x: nextX, y: nextY });
      }
    }

    const route = [];
    let cursor = { x: endPoint.x, y: endPoint.y };
    route.push(cursor);

    while (!(cursor.x === startPoint.x && cursor.y === startPoint.y)) {
      const prior = previous.get(pointKey(cursor.x, cursor.y));
      if (!prior) {
        return [{ x: startPoint.x, y: startPoint.y }, { x: endPoint.x, y: endPoint.y }];
      }

      route.push(prior);
      cursor = prior;
    }

    route.reverse();
    return route;
  }

  function pointKey(x, y) {
    return String(x) + ":" + String(y);
  }

  function countDeadEnds(grid) {
    let total = 0;

    for (let y = 0; y < grid.length; y += 1) {
      for (let x = 0; x < grid[y].length; x += 1) {
        const cell = grid[y][x];
        let walls = 0;
        for (let i = 0; i < wallOrder.length; i += 1) {
          if (cell.walls[wallOrder[i]]) {
            walls += 1;
          }
        }
        if (walls === 3) {
          total += 1;
        }
      }
    }

    return total;
  }

  function renderAnswerButtons(renderModel) {
    answerButtonsNode.innerHTML = "";

    renderModel.exits.forEach(function (exit) {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "answer-button";
      button.innerHTML =
        '<span class="answer-button__label">' + exit.label + "</span>" +
        '<span class="answer-button__meta">Выход</span>';
      button.addEventListener("click", function () {
        handleAnswer(exit.label);
      });
      answerButtonsNode.appendChild(button);
    });
  }

  function handleAnswer(label) {
    if (!currentRender || isResolvingAnswer) {
      return;
    }

    isResolvingAnswer = true;
    window.clearTimeout(regenerateTimerId);

    const isCorrect = label === currentRender.correctExit.label;
    currentRender.feedbackState = isCorrect ? "success" : "error";
    currentRender.revealProgress = 0;

    frame.classList.remove("is-success", "is-error");
    frame.classList.add(isCorrect ? "is-success" : "is-error");

    markAnswerButtons(label, currentRender.correctExit.label, isCorrect);
    setStatus(isCorrect ? "Верно. Показываю маршрут до правильного выхода." : "Неверно. Показываю правильный маршрут.");
    animateReveal(currentRender, function () {
      regenerateTimerId = window.setTimeout(function () {
        controls.seed.value = createSeed();
        generateMazeFromForm({ announce: "Следующий лабиринт готов. Выбери выход." });
      }, REGENERATE_DELAY);
    });
  }

  function markAnswerButtons(selectedLabel, correctLabel, isCorrect) {
    Array.prototype.forEach.call(answerButtonsNode.children, function (button) {
      const labelNode = button.querySelector(".answer-button__label");
      const buttonLabel = labelNode ? labelNode.textContent : "";
      button.disabled = true;
      button.classList.remove("is-correct", "is-wrong");

      if (buttonLabel === correctLabel) {
        button.classList.add("is-correct");
      }

      if (!isCorrect && buttonLabel === selectedLabel) {
        button.classList.add("is-wrong");
      }
    });
  }

  function animateReveal(renderModel, onDone) {
    cancelAnimationFrame(animationFrameId);
    const startedAt = performance.now();

    function frameStep(timestamp) {
      const progress = Math.min(1, (timestamp - startedAt) / REVEAL_DURATION);
      const eased = 1 - Math.pow(1 - progress, 3);
      renderModel.revealProgress = eased;
      drawMaze(renderModel, eased, renderModel.feedbackState);

      if (progress < 1) {
        animationFrameId = requestAnimationFrame(frameStep);
        return;
      }

      if (typeof onDone === "function") {
        onDone();
      }
    }

    animationFrameId = requestAnimationFrame(frameStep);
  }

  function drawMaze(renderModel, routeProgress, feedbackState) {
    const padding = 42;
    const cellSize = renderModel.settings.cellSize;
    const width = renderModel.settings.width;
    const height = renderModel.settings.height;
    const pixelWidth = renderModel.pixelWidth;
    const pixelHeight = renderModel.pixelHeight;
    const totalWidth = pixelWidth + padding * 2;
    const totalHeight = pixelHeight + padding * 2;
    const dpr = Math.max(1, window.devicePixelRatio || 1);

    canvas.width = Math.floor(totalWidth * dpr);
    canvas.height = Math.floor(totalHeight * dpr);
    canvas.style.width = String(totalWidth) + "px";
    canvas.style.height = String(totalHeight) + "px";

    context.setTransform(dpr, 0, 0, dpr, 0, 0);
    context.clearRect(0, 0, totalWidth, totalHeight);

    const innerX = padding;
    const innerY = padding;
    const innerW = pixelWidth;
    const innerH = pixelHeight;
    const wallWidth = Math.max(2, Math.round(cellSize * 0.12));

    drawBackground(totalWidth, totalHeight, innerX, innerY, innerW, innerH, feedbackState);
    if (renderModel.settings.showGrid) {
      drawGrid(innerX, innerY, width, height, cellSize);
    }
    drawExitHighlights(renderModel, innerX, innerY, cellSize);
    if (routeProgress > 0) {
      drawRoute(renderModel.route, innerX, innerY, cellSize, routeProgress, feedbackState);
    }
    drawWalls(renderModel.grid, innerX, innerY, cellSize, wallWidth);
    drawEntrance(renderModel.entrance, innerX, innerY, cellSize);
    drawExitLabels(renderModel.exits, renderModel.correctExit, innerX, innerY, cellSize);
  }

  function drawBackground(totalWidth, totalHeight, innerX, innerY, innerW, innerH, feedbackState) {
    const outer = context.createLinearGradient(0, 0, totalWidth, totalHeight);

    if (feedbackState === "success") {
      outer.addColorStop(0, "#0d3021");
      outer.addColorStop(1, "#1f5b34");
    } else if (feedbackState === "error") {
      outer.addColorStop(0, "#4a1f16");
      outer.addColorStop(1, "#7c2d12");
    } else {
      outer.addColorStop(0, "#132221");
      outer.addColorStop(1, "#243836");
    }

    context.fillStyle = outer;
    roundRect(context, 0, 0, totalWidth, totalHeight, 24);
    context.fill();

    const inner = context.createLinearGradient(innerX, innerY, innerX + innerW, innerY + innerH);
    inner.addColorStop(0, "#fbf3e4");
    inner.addColorStop(1, "#efe6d5");
    context.fillStyle = inner;
    roundRect(context, innerX, innerY, innerW, innerH, 18);
    context.fill();
  }

  function drawGrid(innerX, innerY, width, height, cellSize) {
    context.save();
    context.strokeStyle = "rgba(18, 33, 27, 0.07)";
    context.lineWidth = 1;

    for (let x = 0; x <= width; x += 1) {
      const drawX = innerX + x * cellSize;
      context.beginPath();
      context.moveTo(drawX, innerY);
      context.lineTo(drawX, innerY + height * cellSize);
      context.stroke();
    }

    for (let y = 0; y <= height; y += 1) {
      const drawY = innerY + y * cellSize;
      context.beginPath();
      context.moveTo(innerX, drawY);
      context.lineTo(innerX + width * cellSize, drawY);
      context.stroke();
    }

    context.restore();
  }

  function drawExitHighlights(renderModel, innerX, innerY, cellSize) {
    context.fillStyle = "rgba(32, 49, 44, 0.08)";
    renderModel.exits.forEach(function (exit) {
      context.fillRect(
        innerX + exit.x * cellSize + 2,
        innerY + exit.y * cellSize + 2,
        cellSize - 4,
        cellSize - 4
      );
    });

    context.fillStyle = "rgba(15, 118, 110, 0.18)";
    context.fillRect(
      innerX + renderModel.entrance.x * cellSize + 2,
      innerY + renderModel.entrance.y * cellSize + 2,
      cellSize - 4,
      cellSize - 4
    );
  }

  function drawRoute(route, innerX, innerY, cellSize, progress, feedbackState) {
    if (route.length < 2 || progress <= 0) {
      return;
    }

    const clamped = Math.max(0, Math.min(1, progress));
    const segmentCount = route.length - 1;
    const target = clamped * segmentCount;
    const fullSegments = Math.floor(target);
    const partial = target - fullSegments;

    context.save();
    context.strokeStyle = feedbackState === "success" ? "#22c55e" : "#ef4444";
    context.lineCap = "round";
    context.lineJoin = "round";
    context.lineWidth = Math.max(4, cellSize * 0.32);

    context.beginPath();
    context.moveTo(
      innerX + route[0].x * cellSize + cellSize / 2,
      innerY + route[0].y * cellSize + cellSize / 2
    );

    for (let index = 1; index <= fullSegments; index += 1) {
      context.lineTo(
        innerX + route[index].x * cellSize + cellSize / 2,
        innerY + route[index].y * cellSize + cellSize / 2
      );
    }

    if (fullSegments < segmentCount && partial > 0) {
      const from = route[fullSegments];
      const to = route[fullSegments + 1];
      const startX = innerX + from.x * cellSize + cellSize / 2;
      const startY = innerY + from.y * cellSize + cellSize / 2;
      const endX = innerX + to.x * cellSize + cellSize / 2;
      const endY = innerY + to.y * cellSize + cellSize / 2;
      context.lineTo(
        startX + (endX - startX) * partial,
        startY + (endY - startY) * partial
      );
    }

    context.stroke();
    context.strokeStyle = "rgba(255, 255, 255, 0.32)";
    context.lineWidth = Math.max(1.5, cellSize * 0.08);
    context.stroke();
    context.restore();
  }

  function drawWalls(grid, innerX, innerY, cellSize, wallWidth) {
    context.save();
    context.strokeStyle = "#162120";
    context.lineWidth = wallWidth;
    context.lineCap = "round";

    for (let y = 0; y < grid.length; y += 1) {
      for (let x = 0; x < grid[y].length; x += 1) {
        const cell = grid[y][x];
        const left = innerX + x * cellSize;
        const top = innerY + y * cellSize;
        const right = left + cellSize;
        const bottom = top + cellSize;

        if (cell.walls.north) {
          line(left, top, right, top);
        }
        if (cell.walls.east) {
          line(right, top, right, bottom);
        }
        if (cell.walls.south) {
          line(left, bottom, right, bottom);
        }
        if (cell.walls.west) {
          line(left, top, left, bottom);
        }
      }
    }

    context.restore();
  }

  function drawEntrance(entrance, innerX, innerY, cellSize) {
    const position = exitMarkerPosition(entrance, innerX, innerY, cellSize);
    context.save();
    context.fillStyle = "#0f766e";
    context.beginPath();
    context.arc(position.x, position.y, Math.max(13, cellSize * 0.4), 0, Math.PI * 2);
    context.fill();
    context.fillStyle = "#f7f4ea";
    context.textAlign = "center";
    context.textBaseline = "middle";
    context.font = "700 " + String(Math.max(10, Math.round(cellSize * 0.3))) + "px Aptos, Trebuchet MS, sans-serif";
    context.fillText("IN", position.x, position.y + 1);
    context.restore();
  }

  function drawExitLabels(exits, correctExit, innerX, innerY, cellSize) {
    context.save();
    context.textAlign = "center";
    context.textBaseline = "middle";
    context.font = "700 " + String(Math.max(11, Math.round(cellSize * 0.42))) + "px Aptos, Trebuchet MS, sans-serif";

    exits.forEach(function (exit) {
      const position = exitMarkerPosition(exit, innerX, innerY, cellSize);
      context.fillStyle = exit.label === correctExit.label ? "#20312c" : "#32453f";
      context.beginPath();
      context.arc(position.x, position.y, Math.max(11, cellSize * 0.35), 0, Math.PI * 2);
      context.fill();

      context.fillStyle = "#f7f4ea";
      context.fillText(exit.label, position.x, position.y + 1);
    });

    context.restore();
  }

  function exitMarkerPosition(exit, innerX, innerY, cellSize) {
    const cellCenterX = innerX + exit.x * cellSize + cellSize / 2;
    const cellCenterY = innerY + exit.y * cellSize + cellSize / 2;
    const offset = cellSize * 0.58;

    if (exit.side === "north") {
      return { x: cellCenterX, y: cellCenterY - offset };
    }
    if (exit.side === "south") {
      return { x: cellCenterX, y: cellCenterY + offset };
    }
    if (exit.side === "west") {
      return { x: cellCenterX - offset, y: cellCenterY };
    }

    return { x: cellCenterX + offset, y: cellCenterY };
  }

  function line(x1, y1, x2, y2) {
    context.beginPath();
    context.moveTo(x1, y1);
    context.lineTo(x2, y2);
    context.stroke();
  }

  function roundRect(ctx, x, y, width, height, radius) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.arcTo(x + width, y, x + width, y + height, radius);
    ctx.arcTo(x + width, y + height, x, y + height, radius);
    ctx.arcTo(x, y + height, x, y, radius);
    ctx.arcTo(x, y, x + width, y, radius);
    ctx.closePath();
  }

  function updateSummary(renderModel) {
    metricSeed.textContent = trimSeed(renderModel.settings.seed);
    metricRoute.textContent = String(Math.max(0, renderModel.route.length - 1)) + " steps";
    metricDeadEnds.textContent = String(renderModel.deadEnds);
    metricEntrance.textContent = renderModel.entrance.label;
    mazeSizeNode.textContent = String(renderModel.settings.width) + " x " + String(renderModel.settings.height) + " cells";
  }

  function trimSeed(seed) {
    if (seed.length <= 18) {
      return seed;
    }

    return seed.slice(0, 18) + "...";
  }

  function writeHash(settings) {
    const query = new URLSearchParams();
    query.set("w", String(settings.width));
    query.set("h", String(settings.height));
    query.set("c", String(settings.cellSize));
    query.set("s", settings.seed);
    query.set("grid", settings.showGrid ? "1" : "0");
    window.history.replaceState(null, "", "#" + query.toString());
  }

  function copyShareLink() {
    persistCurrentSettings();
    const url = window.location.href;

    if (!navigator.clipboard || !navigator.clipboard.writeText) {
      setStatus("Clipboard API недоступен в этом браузере.");
      return;
    }

    navigator.clipboard.writeText(url).then(function () {
      setStatus("Ссылка скопирована.");
    }).catch(function () {
      setStatus("Не удалось скопировать ссылку.");
    });
  }

  function downloadPng() {
    if (!currentRender) {
      return;
    }

    const seedSlug = currentRender.settings.seed.replace(/[^a-z0-9_-]+/gi, "-").replace(/^-+|-+$/g, "") || "maze";
    const fileName = "maze-" + seedSlug + ".png";

    if (canvas.toBlob) {
      canvas.toBlob(function (blob) {
        if (!blob) {
          setStatus("Не удалось экспортировать PNG.");
          return;
        }

        const url = URL.createObjectURL(blob);
        triggerDownload(url, fileName);
        window.setTimeout(function () {
          URL.revokeObjectURL(url);
        }, 1000);
        setStatus("PNG сохранён.");
      });
      return;
    }

    triggerDownload(canvas.toDataURL("image/png"), fileName);
    setStatus("PNG сохранён.");
  }

  function triggerDownload(url, fileName) {
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = fileName;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  }

  function setStatus(message) {
    statusNode.textContent = message;
  }

  function persistCurrentSettings() {
    if (!currentRender) {
      return;
    }

    writeHash(currentRender.settings);
    writeStorage(currentRender.settings);
  }

  function createSeed() {
    if (window.crypto && window.crypto.getRandomValues) {
      const values = new Uint32Array(2);
      window.crypto.getRandomValues(values);
      return values[0].toString(36) + "-" + values[1].toString(36);
    }

    return Date.now().toString(36);
  }

  function createRng(seed) {
    const seedGen = xmur3(seed);
    return sfc32(seedGen(), seedGen(), seedGen(), seedGen());
  }

  function xmur3(str) {
    let hash = 1779033703 ^ str.length;
    for (let index = 0; index < str.length; index += 1) {
      hash = Math.imul(hash ^ str.charCodeAt(index), 3432918353);
      hash = (hash << 13) | (hash >>> 19);
    }

    return function () {
      hash = Math.imul(hash ^ (hash >>> 16), 2246822507);
      hash = Math.imul(hash ^ (hash >>> 13), 3266489909);
      hash ^= hash >>> 16;
      return hash >>> 0;
    };
  }

  function sfc32(a, b, c, d) {
    return function () {
      a >>>= 0;
      b >>>= 0;
      c >>>= 0;
      d >>>= 0;
      const value = (a + b) | 0;
      a = b ^ (b >>> 9);
      b = (c + (c << 3)) | 0;
      c = (c << 21) | (c >>> 11);
      d = (d + 1) | 0;
      const result = (value + d) | 0;
      c = (c + result) | 0;
      return (result >>> 0) / 4294967296;
    };
  }
}());
