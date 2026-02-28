const { app, BrowserWindow, screen, ipcMain } = require("electron");
const path = require("path");

app.disableHardwareAcceleration();

function getDisplayData() {
  const displays = screen.getAllDisplays();
  const primary = screen.getPrimaryDisplay();

  return displays.map((d, i) => ({
    index: i,
    label: d.label || `Display ${i}`,
    id: d.id,
    isPrimary: d.id === primary.id,
    bounds: d.bounds,
    size: d.size,
    workArea: d.workArea,
    workAreaSize: d.workAreaSize,
    scaleFactor: d.scaleFactor,
    rotation: d.rotation,
    internal: d.internal,
    colorSpace: d.colorSpace,
    depthPerComponent: d.depthPerComponent,
    colorDepth: d.colorDepth,
  }));
}

function printToConsole(displays) {
  console.log("========================================");
  console.log("Electron Display Info");
  console.log("Electron version:", process.versions.electron);
  console.log("Chromium version:", process.versions.chrome);
  console.log("========================================");
  console.log(`\nTotal displays: ${displays.length}`);
  console.log("----------------------------------------");

  for (const d of displays) {
    console.log(`Display #${d.index} (${d.label})`);
    console.log(`  id:            ${d.id}`);
    console.log(`  isPrimary:     ${d.isPrimary}`);
    console.log(`  bounds:        ${JSON.stringify(d.bounds)}`);
    console.log(`  size:          ${JSON.stringify(d.size)}`);
    console.log(`  workArea:      ${JSON.stringify(d.workArea)}`);
    console.log(`  workAreaSize:  ${JSON.stringify(d.workAreaSize)}`);
    console.log(`  scaleFactor:   ${d.scaleFactor}`);
    console.log(`  rotation:      ${d.rotation}`);
    console.log(`  internal:      ${d.internal}`);
    console.log(`  colorDepth:    ${d.colorDepth}`);
    console.log("----------------------------------------");
  }
}

app.whenReady().then(() => {
  const displays = getDisplayData();
  printToConsole(displays);

  const win = new BrowserWindow({
    width: 720,
    height: 560,
    title: "DeskAct - Electron Display Info",
    webPreferences: {
      contextIsolation: false,
      nodeIntegration: true,
    },
  });

  win.loadFile(path.join(__dirname, "index.html"));

  ipcMain.handle("get-display-data", () => ({
    displays,
    electron: process.versions.electron,
    chrome: process.versions.chrome,
    platform: process.platform,
    arch: process.arch,
  }));
});

app.on("window-all-closed", () => app.quit());
