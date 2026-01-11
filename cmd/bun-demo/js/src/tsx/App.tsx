import { Fragment, jsx } from "./runtime";

export type AppProps = {
  title: string;
  items: string[];
};

export function App(props: AppProps) {
  return (
    <main>
      <header>
        <h1>{props.title}</h1>
        <p>Rendered from TSX inside Goja.</p>
      </header>
      <section>
        <ul>
          {props.items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      </section>
    </main>
  );
}
