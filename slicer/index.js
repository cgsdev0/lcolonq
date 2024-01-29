const Aseprite = require("ase-parser");
const fs = require("fs");
const sharp = require("sharp");

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

  console.log(cels);
}

makePNG();
