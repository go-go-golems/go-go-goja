// ============================================================
// Sample 1: Simple math utilities — minimal doc sentinels
// ============================================================

__package__({
  name: "math/core",
  title: "Core Math Utilities",
  category: "Mathematics",
  guide: "docs/guides/math-overview.md",
  version: "1.0.0",
  description: "Fundamental numeric operations used throughout the library.",
});

// ---- clamp ----

__doc__("clamp", {
  summary: "Clamps a number to the inclusive range [min, max].",
  concepts: ["clamping", "range-restriction"],
  docpage: "docs/math/clamp.md",
  params: [
    { name: "value", type: "number", description: "The input value." },
    { name: "min",   type: "number", description: "Lower bound (inclusive)." },
    { name: "max",   type: "number", description: "Upper bound (inclusive)." },
  ],
  returns: { type: "number", description: "value constrained to [min, max]." },
  related: ["lerp", "saturate"],
  tags: ["math", "core"],
});
export function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

__example__({
  id: "clamp-basic",
  title: "Basic clamping",
  symbols: ["clamp"],
  concepts: ["clamping"],
  tags: ["beginner"],
});
function example_clampBasic() {
  console.assert(clamp(5, 0, 10)  === 5);   // within range — unchanged
  console.assert(clamp(-3, 0, 10) === 0);   // below min  — returns min
  console.assert(clamp(15, 0, 10) === 10);  // above max  — returns max
}

// ---- lerp ----

__doc__("lerp", {
  summary: "Linearly interpolates between a and b by factor t.",
  concepts: ["linear-interpolation"],
  docpage: "docs/math/lerp.md",
  params: [
    { name: "a", type: "number", description: "Start value." },
    { name: "b", type: "number", description: "End value." },
    { name: "t", type: "number", description: "Interpolation factor (0–1)." },
  ],
  returns: { type: "number" },
  related: ["clamp", "smoothstep", "inverseLerp"],
  tags: ["math", "interpolation"],
});
export function lerp(a, b, t) {
  return a + (b - a) * t;
}

__example__({
  id: "lerp-midpoint",
  title: "Finding the midpoint",
  symbols: ["lerp"],
  concepts: ["linear-interpolation"],
  tags: ["beginner"],
});
function example_lerpMidpoint() {
  console.assert(lerp(0, 100, 0.5) === 50);
  console.assert(lerp(0, 100, 0)   === 0);
  console.assert(lerp(0, 100, 1)   === 100);
}

__example__({
  id: "lerp-animation",
  title: "Smooth position animation",
  symbols: ["lerp", "clamp"],
  concepts: ["linear-interpolation", "animation"],
  docpage: "docs/guides/animation.md#lerp",
  tags: ["intermediate"],
});
function example_lerpAnimation() {
  // Move an object from x=0 to x=500 over 2 seconds
  const duration = 2000;
  function getPosition(elapsed) {
    const t = clamp(elapsed / duration, 0, 1);
    return lerp(0, 500, t);
  }
  console.assert(getPosition(1000) === 250);
  console.assert(getPosition(3000) === 500); // clamped at end
}

// ---- saturate ----

__doc__("saturate", {
  summary: "Clamps a value to [0, 1]. Equivalent to clamp(v, 0, 1).",
  concepts: ["clamping", "normalisation"],
  docpage: "docs/math/saturate.md",
  params: [
    { name: "v", type: "number", description: "Input value." },
  ],
  returns: { type: "number", description: "Value in [0, 1]." },
  related: ["clamp"],
  tags: ["math", "core"],
});
export function saturate(v) {
  return clamp(v, 0, 1);
}

// ---- NEW: remap function added to test live reload ----

__doc__("remap", {
  summary: "Re-maps a value from one range [inMin, inMax] to another [outMin, outMax].",
  concepts: ["range-mapping", "linear-interpolation"],
  docpage: "docs/math/remap.md",
  params: [
    { name: "value",  type: "number", description: "Input value." },
    { name: "inMin",  type: "number", description: "Input range lower bound." },
    { name: "inMax",  type: "number", description: "Input range upper bound." },
    { name: "outMin", type: "number", description: "Output range lower bound." },
    { name: "outMax", type: "number", description: "Output range upper bound." },
  ],
  returns: { type: "number", description: "Value mapped to the output range." },
  related: ["lerp", "clamp"],
  tags: ["math", "core"],
});

export function remap(value, inMin, inMax, outMin, outMax) {
  return lerp(outMin, outMax, (value - inMin) / (inMax - inMin));
}

__example__({
  id: "remap-basic",
  title: "Remapping a sensor reading to a display range",
  symbols: ["remap"],
  concepts: ["range-mapping"],
  tags: ["beginner"],
});
function example_remapBasic() {
  // Map a temperature sensor [0, 100] to a gauge [0, 360] degrees
  const angle = remap(25, 0, 100, 0, 360); // => 90
  console.assert(angle === 90);
}
