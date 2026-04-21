export function HowToReadSection() {
  return (
    <section className="essay-article__section">
      <h2>How To Read This Section</h2>
      <p className="essay-prose">
        If you are new to the system, do not treat the button and tables as isolated widgets.
        Treat them as synchronized views over one concrete backend event: the creation of a REPL
        session. The prose explains why the event matters, the summary card shows the most
        important fields, the policy card shows the behavior contract, and the JSON panel proves
        the page is not inventing simplified data.
      </p>
      <p className="essay-prose">
        A useful way to study the system is to read the page in three passes. First, build the
        mental model of what a session is. Second, follow the request path from browser to handler
        to session service to persistence. Third, compare the prose with the actual JSON until the
        page feels mechanically understandable rather than magical.
      </p>
      <ul className="essay-detail-list">
        <li>
          <strong>Identity.</strong> A session ID is the handle you will use to refer to one
          long-lived unit of REPL state.
        </li>
        <li>
          <strong>Profile.</strong> A profile selects a preset behavior shape such as persistent or
          raw.
        </li>
        <li>
          <strong>Policy.</strong> Policy is the executable contract that determines evaluation,
          observation, and persistence behavior.
        </li>
        <li>
          <strong>Snapshot.</strong> A snapshot is a read view over the current session state at a
          particular moment.
        </li>
      </ul>
    </section>
  );
}
