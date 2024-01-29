const Aseprite = require("ase-parser");
const fs = require("fs");

async function makePNG() {
  const buff = fs.readFileSync("../levels.ase");
  const ase = new Aseprite(buff, "levels.ase");

  ase.parse();

  // Get the cels for the first frame
  const cels = ase.frames[0].cels
    // copy the array
    .map((a) => a)
    .sort((a, b) => {
      const orderA = a.layerIndex + a.zIndex;
      const orderB = b.layerIndex + b.zIndex;
      // sort by order, then by zIndex
      return orderA - orderB || a.zIndex - b.zIndex;
    });

  const { w, h, rawCelData } = cels[0];
  for (let gr = 0; gr < 10; gr++) {
    for (let gc = 0; gc < 10; gc++) {
      const filename = `../map/${gr}x${gc}.txt`;
      console.log(filename);
      let data = "";
      for (let ir = 0; ir < h / 10; ir++) {
        for (let ic = 0; ic < w / 10; ic++) {
          const row = (gr * h) / 10 + ir;
          const col = (gc * w) / 10 + ic;
          const i = (row * w + col) * 4;
          const r = rawCelData.readUint8(i);
          const g = rawCelData.readUint8(i + 1);
          const b = rawCelData.readUint8(i + 2);
          const a = rawCelData.readUint8(i + 3);
          if (a === 0) {
            data += ".";
          } else {
            if (r === 0 && g === 0 && b === 255) {
              data += "S";
            } else if (r === 255 && g === 255 && b === 0) {
              data += "$";
            } else if (r === 255 && g === 0 && b === 255) {
              data += "x";
            } else if (r > 0 && g === 0 && b === 0) {
              data += ["b"][255 - r];
            } else {
              data += "#";
            }
          }
        }
        data += "\n";
      }
      data += `${gr - 1}x${gc}\n`;
      data += `${gr}x${gc + 1}\n`;
      data += `${gr + 1}x${gc}\n`;
      data += `${gr}x${gc - 1}\n`;
      fs.writeFileSync(filename, data);
    }
  }
}

makePNG();
