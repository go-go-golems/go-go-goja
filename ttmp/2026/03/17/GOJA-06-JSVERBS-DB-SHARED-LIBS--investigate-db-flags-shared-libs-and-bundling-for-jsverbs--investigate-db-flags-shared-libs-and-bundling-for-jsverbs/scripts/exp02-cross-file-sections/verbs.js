__package__({
  name: "cross-file",
  short: "Cross-file section reference demo"
});

function probe(dbConfig) {
  return {
    db: dbConfig.db
  };
}

__verb__("probe", {
  short: "Intentionally references a section declared in another file",
  sections: ["db"],
  fields: {
    dbConfig: {
      bind: "db"
    }
  }
});
