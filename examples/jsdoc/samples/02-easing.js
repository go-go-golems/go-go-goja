// ============================================================
// Sample 2: Easing functions — medium complexity, uses tagged
//           template literals for long-form prose docs
// ============================================================

__package__({
  name: "animation/easing",
  title: "Easing Functions",
  category: "Animation",
  guide: "docs/guides/animation.md",
  version: "1.2.0",
  description: "A collection of standard easing functions for animation curves.",
  seeAlso: ["math/core", "animation/tween"],
});

// ---- smoothstep ----

__doc__("smoothstep", {
  summary: "Hermite interpolation with smooth start and end (Ken Perlin).",
  concepts: ["easing", "smoothstep", "hermite-interpolation"],
  docpage: "docs/animation/smoothstep.md",
  params: [
    { name: "edge0", type: "number", description: "Lower edge of the transition." },
    { name: "edge1", type: "number", description: "Upper edge of the transition." },
    { name: "x",     type: "number", description: "Input value." },
  ],
  returns: { type: "number", description: "Smoothly interpolated value in [0, 1]." },
  related: ["smootherstep", "lerp", "clamp"],
  tags: ["easing", "animation", "core"],
});
doc`
---
symbol: smoothstep
---

**smoothstep** produces a smooth Hermite interpolation between 0 and 1 when
\`x\` is in the range [\`edge0\`, \`edge1\`]. It is equivalent to the GLSL built-in
of the same name and is widely used in shader programming and animation.

The polynomial used is **3t² − 2t³**, which has zero first-derivative at both
endpoints, giving a smooth S-curve with no sudden velocity changes.

## Formula

    t = clamp((x - edge0) / (edge1 - edge0), 0, 1)
    result = t * t * (3 - 2 * t)

## Notes

- When \`x ≤ edge0\` the result is 0.
- When \`x ≥ edge1\` the result is 1.
- For an even smoother curve with zero second-derivative at endpoints, see
  \`smootherstep\`.
`;
export function smoothstep(edge0, edge1, x) {
  const t = Math.min(Math.max((x - edge0) / (edge1 - edge0), 0), 1);
  return t * t * (3 - 2 * t);
}

__example__({
  id: "smoothstep-basic",
  title: "Basic smoothstep usage",
  symbols: ["smoothstep"],
  concepts: ["easing", "smoothstep"],
  tags: ["beginner"],
});
function example_smoothstepBasic() {
  console.assert(smoothstep(0, 1, 0)   === 0);
  console.assert(smoothstep(0, 1, 0.5) === 0.5);
  console.assert(smoothstep(0, 1, 1)   === 1);
  // Outside range → clamped
  console.assert(smoothstep(0, 1, -1)  === 0);
  console.assert(smoothstep(0, 1, 2)   === 1);
}

__example__({
  id: "smoothstep-fade",
  title: "Fade-in effect using smoothstep",
  symbols: ["smoothstep"],
  concepts: ["easing", "animation"],
  docpage: "docs/guides/animation.md#fade",
  tags: ["intermediate"],
});
function example_smoothstepFade() {
  // Fade opacity from 0 to 1 between t=0.2 and t=0.8
  function fadeOpacity(t) {
    return smoothstep(0.2, 0.8, t);
  }
  console.assert(fadeOpacity(0)   === 0);
  console.assert(fadeOpacity(0.5) === 0.5);
  console.assert(fadeOpacity(1)   === 1);
}

// ---- smootherstep ----

__doc__("smootherstep", {
  summary: "Ken Perlin's improved smoothstep with zero second derivative at edges.",
  concepts: ["easing", "smootherstep", "hermite-interpolation"],
  docpage: "docs/animation/smootherstep.md",
  params: [
    { name: "edge0", type: "number", description: "Lower edge." },
    { name: "edge1", type: "number", description: "Upper edge." },
    { name: "x",     type: "number", description: "Input value." },
  ],
  returns: { type: "number" },
  related: ["smoothstep"],
  tags: ["easing", "animation"],
});
doc`
---
symbol: smootherstep
---

**smootherstep** is Ken Perlin's improvement over \`smoothstep\`. It uses the
polynomial **6t⁵ − 15t⁴ + 10t³**, which has zero first *and* second derivatives
at both endpoints. This eliminates the visible "jerk" that can occur when
concatenating multiple smoothstep segments.

Prefer \`smootherstep\` when you need C² continuity (e.g., procedural noise,
high-quality camera transitions).
`;
export function smootherstep(edge0, edge1, x) {
  const t = Math.min(Math.max((x - edge0) / (edge1 - edge0), 0), 1);
  return t * t * t * (t * (t * 6 - 15) + 10);
}

// ---- easeInQuad / easeOutQuad / easeInOutQuad ----

__doc__("easeInQuad", {
  summary: "Quadratic ease-in: accelerates from zero velocity.",
  concepts: ["easing", "quadratic"],
  docpage: "docs/animation/easing-curves.md",
  params: [{ name: "t", type: "number", description: "Normalised time [0, 1]." }],
  returns: { type: "number" },
  related: ["easeOutQuad", "easeInOutQuad"],
  tags: ["easing"],
});
export const easeInQuad = t => t * t;

__doc__("easeOutQuad", {
  summary: "Quadratic ease-out: decelerates to zero velocity.",
  concepts: ["easing", "quadratic"],
  docpage: "docs/animation/easing-curves.md",
  params: [{ name: "t", type: "number", description: "Normalised time [0, 1]." }],
  returns: { type: "number" },
  related: ["easeInQuad", "easeInOutQuad"],
  tags: ["easing"],
});
export const easeOutQuad = t => t * (2 - t);

__doc__("easeInOutQuad", {
  summary: "Quadratic ease-in-out: accelerates then decelerates.",
  concepts: ["easing", "quadratic"],
  docpage: "docs/animation/easing-curves.md",
  params: [{ name: "t", type: "number", description: "Normalised time [0, 1]." }],
  returns: { type: "number" },
  related: ["easeInQuad", "easeOutQuad", "smoothstep"],
  tags: ["easing"],
});
export const easeInOutQuad = t =>
  t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t;

__example__({
  id: "easing-comparison",
  title: "Comparing quadratic easing variants at t=0.5",
  symbols: ["easeInQuad", "easeOutQuad", "easeInOutQuad"],
  concepts: ["easing", "quadratic"],
  tags: ["intermediate"],
});
function example_easingComparison() {
  const t = 0.5;
  console.log("easeInQuad   :", easeInQuad(t));    // 0.25
  console.log("easeOutQuad  :", easeOutQuad(t));   // 0.75
  console.log("easeInOutQuad:", easeInOutQuad(t)); // 0.5
}
