// ============================================================
// Sample 3: 2D Vector class — complex, class methods, rich docs
// All __doc__ sentinels are at top level (before the class),
// which is the correct pattern for class symbols.
// ============================================================

__package__({
  name: "math/vector2",
  title: "2D Vector Mathematics",
  category: "Mathematics",
  guide: "docs/guides/vectors.md",
  version: "2.0.0",
  description: "Immutable 2D vector type with a full suite of geometric operations.",
  seeAlso: ["math/core", "math/matrix2"],
});

doc`
---
package: math/vector2
---

# 2D Vector Mathematics

This module provides the \`Vec2\` class — an immutable, chainable 2D vector type
designed for use in game engines, physics simulations, and generative graphics.

All methods return **new** \`Vec2\` instances; the original is never mutated.
This makes \`Vec2\` safe to use in functional pipelines and React state.

## Coordinate System

The library uses a standard right-handed coordinate system:

- **+x** points right
- **+y** points up

When working with HTML Canvas or CSS transforms, remember that Canvas uses
**+y downward** — you may need to negate the y component.

## Performance Notes

For hot paths (e.g., physics loops updating thousands of bodies per frame),
consider using plain \`{x, y}\` objects and inline arithmetic instead of \`Vec2\`
instances to avoid allocation pressure.
`;

// ---- Vec2 class documentation ----

__doc__("Vec2", {
  summary: "Immutable 2D vector with chainable geometric operations.",
  concepts: ["vector-math", "2d-geometry", "linear-algebra"],
  docpage: "docs/math/vec2.md",
  params: [
    { name: "x", type: "number", description: "Horizontal component." },
    { name: "y", type: "number", description: "Vertical component." },
  ],
  related: ["Vec2.add", "Vec2.scale", "Vec2.dot", "Vec2.normalize", "Vec2.lerp"],
  tags: ["math", "geometry", "core"],
});

__doc__("Vec2.zero", {
  summary: "Returns the zero vector (0, 0).",
  concepts: ["vector-math"],
  tags: ["factory"],
});

__doc__("Vec2.one", {
  summary: "Returns the unit vector (1, 1).",
  concepts: ["vector-math"],
  tags: ["factory"],
});

__doc__("Vec2.fromAngle", {
  summary: "Creates a unit vector from an angle in radians.",
  concepts: ["vector-math", "polar-coordinates"],
  params: [{ name: "radians", type: "number", description: "Angle from +x axis." }],
  returns: { type: "Vec2", description: "Unit vector pointing in the given direction." },
  related: ["Vec2.angle"],
  tags: ["factory", "geometry"],
});

__doc__("Vec2.add", {
  summary: "Returns the vector sum of this and another vector.",
  concepts: ["vector-math", "vector-addition"],
  params: [{ name: "other", type: "Vec2", description: "Vector to add." }],
  returns: { type: "Vec2" },
  related: ["Vec2.sub", "Vec2.scale"],
  tags: ["arithmetic"],
});

__doc__("Vec2.sub", {
  summary: "Returns the vector difference (this − other).",
  concepts: ["vector-math"],
  params: [{ name: "other", type: "Vec2" }],
  returns: { type: "Vec2" },
  related: ["Vec2.add"],
  tags: ["arithmetic"],
});

__doc__("Vec2.scale", {
  summary: "Multiplies both components by a scalar.",
  concepts: ["vector-math", "scalar-multiplication"],
  params: [{ name: "s", type: "number", description: "Scalar factor." }],
  returns: { type: "Vec2" },
  tags: ["arithmetic"],
});

__doc__("Vec2.dot", {
  summary: "Returns the dot product of this and another vector.",
  concepts: ["vector-math", "dot-product", "inner-product"],
  docpage: "docs/math/vec2.md#dot-product",
  params: [{ name: "other", type: "Vec2" }],
  returns: { type: "number", description: "Scalar dot product." },
  related: ["Vec2.cross", "Vec2.angle"],
  tags: ["geometry"],
});

__doc__("Vec2.cross", {
  summary: "Returns the 2D cross product (scalar z-component of 3D cross).",
  concepts: ["vector-math", "cross-product"],
  params: [{ name: "other", type: "Vec2" }],
  returns: { type: "number", description: "Signed area of the parallelogram." },
  related: ["Vec2.dot"],
  tags: ["geometry"],
});

__doc__("Vec2.length", {
  summary: "Returns the Euclidean length (magnitude) of the vector.",
  concepts: ["vector-math", "magnitude"],
  returns: { type: "number" },
  related: ["Vec2.lengthSq", "Vec2.normalize"],
  tags: ["geometry"],
});

__doc__("Vec2.lengthSq", {
  summary: "Returns the squared length. Cheaper than length when only comparing distances.",
  concepts: ["vector-math", "magnitude"],
  returns: { type: "number" },
  related: ["Vec2.length"],
  tags: ["geometry", "performance"],
});

__doc__("Vec2.normalize", {
  summary: "Returns a unit vector in the same direction. Returns zero vector if length is 0.",
  concepts: ["vector-math", "normalisation"],
  returns: { type: "Vec2" },
  related: ["Vec2.length", "Vec2.fromAngle"],
  tags: ["geometry"],
});

__doc__("Vec2.lerp", {
  summary: "Linearly interpolates between this and another vector by factor t.",
  concepts: ["linear-interpolation", "vector-math"],
  params: [
    { name: "other", type: "Vec2", description: "Target vector." },
    { name: "t",     type: "number", description: "Interpolation factor [0, 1]." },
  ],
  returns: { type: "Vec2" },
  related: ["Vec2.add", "Vec2.scale"],
  tags: ["interpolation", "animation"],
});

__doc__("Vec2.angle", {
  summary: "Returns the angle of this vector from the +x axis, in radians.",
  concepts: ["vector-math", "polar-coordinates"],
  returns: { type: "number", description: "Angle in radians (−π to π)." },
  related: ["Vec2.fromAngle"],
  tags: ["geometry"],
});

// ---- Vec2 class implementation ----

export class Vec2 {
  constructor(x = 0, y = 0) {
    this.x = x;
    this.y = y;
    Object.freeze(this);
  }

  static zero()  { return new Vec2(0, 0); }
  static one()   { return new Vec2(1, 1); }

  static fromAngle(radians) {
    return new Vec2(Math.cos(radians), Math.sin(radians));
  }

  add(other)   { return new Vec2(this.x + other.x, this.y + other.y); }
  sub(other)   { return new Vec2(this.x - other.x, this.y - other.y); }
  scale(s)     { return new Vec2(this.x * s, this.y * s); }
  dot(other)   { return this.x * other.x + this.y * other.y; }
  cross(other) { return this.x * other.y - this.y * other.x; }

  get length()   { return Math.sqrt(this.x * this.x + this.y * this.y); }
  get lengthSq() { return this.x * this.x + this.y * this.y; }

  normalize() {
    const len = this.length;
    return len === 0 ? Vec2.zero() : this.scale(1 / len);
  }

  lerp(other, t) {
    return new Vec2(
      this.x + (other.x - this.x) * t,
      this.y + (other.y - this.y) * t,
    );
  }

  get angle() { return Math.atan2(this.y, this.x); }

  toString() { return `Vec2(${this.x}, ${this.y})`; }
}

// ---- Examples ----

__example__({
  id: "vec2-basic-arithmetic",
  title: "Basic vector arithmetic",
  symbols: ["Vec2", "Vec2.add", "Vec2.sub", "Vec2.scale"],
  concepts: ["vector-math", "vector-addition"],
  tags: ["beginner"],
});
function example_vec2BasicArithmetic() {
  const a = new Vec2(3, 4);
  const b = new Vec2(1, 2);
  console.assert(a.add(b).x === 4 && a.add(b).y === 6);
  console.assert(a.sub(b).x === 2 && a.sub(b).y === 2);
  console.assert(a.scale(2).x === 6 && a.scale(2).y === 8);
}

__example__({
  id: "vec2-normalize-dot",
  title: "Normalisation and dot product",
  symbols: ["Vec2.normalize", "Vec2.dot", "Vec2.length"],
  concepts: ["vector-math", "normalisation", "dot-product"],
  tags: ["intermediate"],
});
function example_vec2NormalizeDot() {
  const v = new Vec2(3, 4);
  const n = v.normalize();
  console.assert(Math.abs(n.length - 1) < 1e-10);
  const right = new Vec2(1, 0);
  const up    = new Vec2(0, 1);
  console.assert(right.dot(up) === 0);
}

__example__({
  id: "vec2-angle-fromAngle",
  title: "Converting between angle and vector",
  symbols: ["Vec2.fromAngle", "Vec2.angle"],
  concepts: ["vector-math", "polar-coordinates"],
  tags: ["intermediate"],
});
function example_vec2AngleRoundtrip() {
  const angle = Math.PI / 4;
  const v = Vec2.fromAngle(angle);
  console.assert(Math.abs(v.angle - angle) < 1e-10);
}

__example__({
  id: "vec2-lerp-path",
  title: "Interpolating along a path between two points",
  symbols: ["Vec2.lerp"],
  concepts: ["linear-interpolation", "vector-math", "animation"],
  docpage: "docs/guides/vectors.md#animation",
  tags: ["intermediate"],
});
function example_vec2LerpPath() {
  const start = new Vec2(0, 0);
  const end   = new Vec2(100, 50);
  const mid   = start.lerp(end, 0.5);
  console.assert(mid.x === 50 && mid.y === 25);
}
