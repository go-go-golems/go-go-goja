import { Heading } from "@/components/essay/Heading";
import { Prose } from "@/components/essay/Prose";

type SectionIntroProps = {
  title: string;
  summary: string;
  introParagraphs: string[];
};

export function SectionIntro({ title, summary, introParagraphs }: SectionIntroProps) {
  return (
    <section className="essay-intro" data-part="section-intro">
      <Heading n={1}>{title}</Heading>
      <Prose>
        {summary}
      </Prose>
      <Prose>
        A session is more than a prompt window. It has an{" "}
        <span className="essay-emph-blue">identity</span>, a{" "}
        <span className="essay-emph-blue">profile</span>, a{" "}
        <span className="essay-emph-blue">policy</span>, and evolving state. In this first
        section, the job is to learn those pieces before moving on to evaluation history, binding
        evolution, or richer runtime behavior.
      </Prose>
      {introParagraphs.map((paragraph) => (
        <Prose key={paragraph}>{paragraph}</Prose>
      ))}
    </section>
  );
}
