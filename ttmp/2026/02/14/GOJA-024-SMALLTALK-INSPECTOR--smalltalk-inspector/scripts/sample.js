const symCustom = Symbol("custom");

const config = {
  apiUrl: "https://api.example.com/v3",
  timeout: 3000,
  retries: 5,
  debug: false,
};

class Animal {
  constructor(name) {
    this.name = name;
    this.alive = true;
    this.energy = 0;
  }

  eat(food) {
    if (!this.alive) {
      throw new Error("dead");
    }
    this.energy += food.calories;
    return this;
  }

  sleep() {
    return "zzz";
  }
}

class Dog extends Animal {
  constructor(name) {
    super(name);
    this.breed = "lab";
  }

  bark() {
    const sound = this.breed === "husky" ? "awoo" : "woof";
    return sound;
  }

  fetch(item) {
    return this.eat(item);
  }
}

function greet(name) {
  return `Hello, ${name}!`;
}

function main() {
  const rex = new Dog("Rex");
  rex.bark();
  return rex;
}

const API_KEY = "sk-abc123";
const version = 3;
