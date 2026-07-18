function openInbox(path) {
  const database = require("database");
  database.configure("sqlite3", path || "personal-inbox.sqlite");
  database.exec(`
    create table if not exists inbox_items (
      id text primary key,
      title text not null,
      url text not null default '',
      note text not null default '',
      source text not null,
      submitted_by_kind text not null,
      submitted_by_id text not null,
      created_at text not null,
      archived_at text not null default ''
    )
  `);
  return database;
}

function insertInboxItem(database, input) {
  const item = {
    id: newItemID(),
    title: String(input.title || "Untitled capture"),
    url: String(input.url || ""),
    note: String(input.note || ""),
    source: String(input.source || "api"),
    submittedByKind: String(input.submittedByKind || "anonymousApi"),
    submittedById: String(input.submittedById || "anonymous"),
    createdAt: new Date().toISOString(),
    archivedAt: ""
  };

  database.exec(
    `insert into inbox_items
      (id, title, url, note, source, submitted_by_kind, submitted_by_id, created_at, archived_at)
     values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    item.id,
    item.title,
    item.url,
    item.note,
    item.source,
    item.submittedByKind,
    item.submittedById,
    item.createdAt,
    item.archivedAt
  );

  return item;
}

function listInboxItems(database, includeArchived) {
  const where = includeArchived ? "" : "where archived_at = ''";
  return queryInboxItems(database, where, []);
}

function listInboxItemsForUser(database, userID, includeArchived) {
  const clauses = ["submitted_by_kind = 'sessionUser'", "submitted_by_id = ?"];
  if (!includeArchived) clauses.push("archived_at = ''");
  return queryInboxItems(database, `where ${clauses.join(" and ")}`, [String(userID || "")]);
}

function queryInboxItems(database, where, args) {
  return database.query(
    `select id,
            title,
            url,
            note,
            source,
            submitted_by_kind as submittedByKind,
            submitted_by_id as submittedById,
            created_at as createdAt,
            archived_at as archivedAt
       from inbox_items
       ${where}
      order by created_at desc, id desc`,
    ...(args || [])
  );
}

function archiveInboxItem(database, id) {
  const archivedAt = new Date().toISOString();
  database.exec(
    "update inbox_items set archived_at = ? where id = ? and archived_at = ''",
    archivedAt,
    id
  );
  return { id, archivedAt };
}

function archiveInboxItemForUser(database, id, userID) {
  const archivedAt = new Date().toISOString();
  database.exec(
    "update inbox_items set archived_at = ? where id = ? and submitted_by_kind = 'sessionUser' and submitted_by_id = ? and archived_at = ''",
    archivedAt,
    id,
    String(userID || "")
  );
  return { id, archivedAt };
}

function newItemID() {
  return `item_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

module.exports = {
  openInbox,
  insertInboxItem,
  listInboxItems,
  listInboxItemsForUser,
  archiveInboxItem,
  archiveInboxItemForUser
};
