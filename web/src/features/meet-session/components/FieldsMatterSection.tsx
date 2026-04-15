export function FieldsMatterSection() {
  return (
    <section className="essay-article__section">
      <h2>Why The Main Fields Matter</h2>
      <p className="essay-prose">
        The summary card is intentionally small. That is a feature, not a limitation. When you are
        learning a new system, the highest-value move is to identify the few fields that will keep
        showing up in every later workflow. In this REPL, those fields are the session ID, the
        selected profile, the creation time, and the running counters for cells and bindings.
      </p>
      <ul className="essay-detail-list">
        <li>
          <code>id</code>: the durable lookup key used for later snapshot, history, and
          export-style operations.
        </li>
        <li>
          <code>profile</code>: the preset that selected the current policy defaults.
        </li>
        <li>
          <code>createdAt</code>: when the backend recorded session creation.
        </li>
        <li>
          <code>cellCount</code>: how many user submissions have been evaluated so far.
        </li>
        <li>
          <code>bindingCount</code>: how many tracked bindings the session currently exposes.
        </li>
      </ul>
      <p className="essay-prose">
        The policy card belongs immediately after the summary card because the two answers belong
        together. The summary tells you <em>what exists now</em>. The policy tells you{" "}
        <em>what kinds of behavior are allowed to happen next</em>. Together they answer both state
        and behavior, which is the minimum needed to reason about any REPL session.
      </p>
    </section>
  );
}
