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
      const metaname = `../meta/${gr}x${gc}.txt`;
      console.log(filename);
      let data = "";
      data += `${gr - 1}x${gc}\n`;
      data += `${gr}x${gc + 1}\n`;
      data += `${gr + 1}x${gc}\n`;
      data += `${gr}x${gc - 1}\n`;
      const buffer = new ArrayBuffer(40 * 16 * 4);
      const view = new Uint8Array(buffer);
      for (let ir = 0; ir < h / 10; ir++) {
        for (let ic = 0; ic < w / 10; ic++) {
          const row = (gr * h) / 10 + ir;
          const col = (gc * w) / 10 + ic;
          const i = (row * w + col) * 4;
          const j = (ir * (w / 10) + ic) * 4;
          const r = rawCelData.readUint8(i);
          const g = rawCelData.readUint8(i + 1);
          const b = rawCelData.readUint8(i + 2);
          const a = rawCelData.readUint8(i + 3);
          view[j] = r;
          view[j + 1] = g;
          view[j + 2] = b;
          view[j + 3] = a;
        }
        buffer.data += "\n";
      }
      fs.writeFileSync(metaname, data);
      fs.writeFileSync(filename, Buffer.from(buffer));
    }
  }
}

makePNG();
