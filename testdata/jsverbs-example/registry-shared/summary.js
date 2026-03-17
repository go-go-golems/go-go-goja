function summarizeFilters(filters) {
  return {
    state: filters.state,
    labelCount: (filters.labels || []).length
  };
}

__verb__("summarizeFilters", {
  short: "Reuse the same host-registered shared section",
  fields: {
    filters: {
      bind: "filters"
    }
  }
});
