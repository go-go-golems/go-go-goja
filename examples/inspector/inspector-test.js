// inspector-test.js — A file for testing the Smalltalk inspector TUI
// Covers: classes, inheritance, functions, constants, symbols, closures, arrays, maps

const VERSION = "1.0.0";
const MAX_ITEMS = 100;

const settings = {
  theme: "dark",
  fontSize: 14,
  lineNumbers: true,
};

class Shape {
  constructor(color) {
    this.color = color;
    this.visible = true;
  }

  describe() {
    return `A ${this.color} shape`;
  }

  hide() {
    this.visible = false;
  }
}

class Circle extends Shape {
  constructor(color, radius) {
    super(color);
    this.radius = radius;
  }

  area() {
    return Math.PI * this.radius * this.radius;
  }

  perimeter() {
    return 2 * Math.PI * this.radius;
  }

  describe() {
    return `A ${this.color} circle with radius ${this.radius}`;
  }
}

class Rectangle extends Shape {
  constructor(color, width, height) {
    super(color);
    this.width = width;
    this.height = height;
  }

  area() {
    return this.width * this.height;
  }

  perimeter() {
    return 2 * (this.width + this.height);
  }
}

function createShape(type, color, ...args) {
  switch (type) {
    case "circle":
      return new Circle(color, args[0] || 1);
    case "rectangle":
      return new Rectangle(color, args[0] || 1, args[1] || 1);
    default:
      return new Shape(color);
  }
}

function totalArea(shapes) {
  let sum = 0;
  for (const s of shapes) {
    if (typeof s.area === "function") {
      sum += s.area();
    }
  }
  return sum;
}

const myCircle = new Circle("red", 5);
const myRect = new Rectangle("blue", 3, 4);
const shapes = [myCircle, myRect];

function main() {
  const c = createShape("circle", "green", 10);
  const r = createShape("rectangle", "yellow", 6, 8);
  return { circle: c, rectangle: r, totalArea: totalArea([c, r]) };
}
