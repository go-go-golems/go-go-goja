function greet(name, excited) {
  return {
    greeting: excited ? `Hello, ${name}!` : `Hello, ${name}`
  };
}

__verb__("greet", {
  short: "Greet from the dedicated botcli fixture",
  fields: {
    name: {
      argument: true,
      help: "Person name"
    },
    excited: {
      type: "bool",
      short: "e",
      help: "Add excitement"
    }
  }
});

function banner(name) {
  return `*** ${name} ***\n`;
}

__verb__("banner", {
  short: "Render plain text from the dedicated botcli fixture",
  output: "text",
  fields: {
    name: {
      argument: true
    }
  }
});
