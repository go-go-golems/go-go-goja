type PolicyRowProps = {
  label: string;
  value: string | number | boolean;
};

export function PolicyRow({ label, value }: PolicyRowProps) {
  const isBool = typeof value === "boolean";
  const toneClass = isBool
    ? value
      ? "essay-policy-row__value--ok"
      : "essay-policy-row__value--off"
    : "essay-policy-row__value--accent";

  const rendered = isBool ? (value ? "✓ yes" : "— no") : String(value);

  return (
    <tr className="essay-policy-row">
      <td>{label}</td>
      <td className={toneClass}>{rendered}</td>
    </tr>
  );
}
