import { fileGuide } from "@/features/meet-session/components/fieldGuideData";

export function SourceFileGuideSection() {
  return (
    <section className="essay-article__section">
      <h2>Source File Guide</h2>
      <p className="essay-prose">
        These are the files worth reading in order once the live demo makes sense.
      </p>
      <table className="essay-reference-table">
        <thead>
          <tr>
            <th>File</th>
            <th>Why it matters</th>
          </tr>
        </thead>
        <tbody>
          {fileGuide.map((file) => (
            <tr key={file.path}>
              <td>
                <code>{file.path}</code>
              </td>
              <td>{file.note}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
