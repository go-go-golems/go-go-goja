__package__({
  name: "inbox",
  short: "Personal Knowledge Inbox tutorial commands"
});

__verb__("hello", {
  name: "hello",
  output: "text",
  short: "Say hello from the first Personal Knowledge Inbox xgoja verb",
  fields: {
    name: {
      type: "string",
      default: "world",
      help: "Name to greet"
    }
  }
});

function hello(name) {
  return `Hello, ${name || "world"}! This is the Personal Knowledge Inbox tutorial.`;
}
