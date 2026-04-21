export function ValidationExercisesSection() {
  return (
    <section className="essay-article__section">
      <h2>Validation Exercises</h2>
      <ol className="essay-detail-list essay-detail-list--ordered">
        <li>Create a session and compare the summary card to the raw JSON panel field by field.</li>
        <li>Confirm that the policy card matches the nested policy object in the JSON payload.</li>
        <li>Reload the page and verify that a stored session ID can be used to fetch a fresh snapshot.</li>
        <li>Open devtools and confirm that the page is calling the article-scoped routes shown above.</li>
        <li>Read the referenced source files and verify that your mental model matches the actual control flow.</li>
      </ol>
      <p className="essay-prose">
        A good intern habit is to move back and forth between the browser, the JSON payload, and
        the code until each explanation can be grounded in one concrete artifact.
      </p>
    </section>
  );
}
