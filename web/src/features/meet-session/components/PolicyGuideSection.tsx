export function PolicyGuideSection() {
  return (
    <section className="essay-article__section">
      <h2>Policy Guide</h2>
      <ul className="essay-detail-list">
        <li>
          <strong>Eval.</strong> Defines how code executes, including mode, timeout, and top-level
          await behavior.
        </li>
        <li>
          <strong>Observe.</strong> Controls which reports and snapshots the system produces, such
          as static analysis and binding tracking.
        </li>
        <li>
          <strong>Persist.</strong> Controls which pieces of session history are written to the
          durable store.
        </li>
      </ul>
      <p className="essay-prose">
        For a new intern, this is the key engineering lesson: never describe REPL behavior only in
        informal prose. The policy object is the precise machine-readable version of that story.
      </p>
    </section>
  );
}
