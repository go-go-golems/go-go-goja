import type { ProfileName, ProfileSpec } from "@/features/meet-session/types";

type ComparisonRow = {
  label: string;
  values: Record<ProfileName, string>;
};

type ProfileComparisonTableProps = {
  profiles: ProfileSpec[];
  selectedProfile: ProfileName;
};

function boolLabel(value: boolean) {
  return value ? "yes" : "no";
}

function buildRows(profiles: ProfileSpec[]): ComparisonRow[] {
  const byID = Object.fromEntries(
    profiles.map((profile) => [profile.id, profile])
  ) as Record<ProfileName, ProfileSpec>;
  return [
    {
      label: "Eval mode",
      values: {
        raw: byID.raw.policy.eval.mode,
        interactive: byID.interactive.policy.eval.mode,
        persistent: byID.persistent.policy.eval.mode
      }
    },
    {
      label: "Capture last expression",
      values: {
        raw: boolLabel(byID.raw.policy.eval.captureLastExpression),
        interactive: boolLabel(byID.interactive.policy.eval.captureLastExpression),
        persistent: boolLabel(byID.persistent.policy.eval.captureLastExpression)
      }
    },
    {
      label: "Top-level await",
      values: {
        raw: boolLabel(byID.raw.policy.eval.supportTopLevelAwait),
        interactive: boolLabel(byID.interactive.policy.eval.supportTopLevelAwait),
        persistent: boolLabel(byID.persistent.policy.eval.supportTopLevelAwait)
      }
    },
    {
      label: "Static analysis",
      values: {
        raw: boolLabel(byID.raw.policy.observe.staticAnalysis),
        interactive: boolLabel(byID.interactive.policy.observe.staticAnalysis),
        persistent: boolLabel(byID.persistent.policy.observe.staticAnalysis)
      }
    },
    {
      label: "Binding tracking",
      values: {
        raw: boolLabel(byID.raw.policy.observe.bindingTracking),
        interactive: boolLabel(byID.interactive.policy.observe.bindingTracking),
        persistent: boolLabel(byID.persistent.policy.observe.bindingTracking)
      }
    },
    {
      label: "Persistence",
      values: {
        raw: boolLabel(byID.raw.policy.persist.enabled),
        interactive: boolLabel(byID.interactive.policy.persist.enabled),
        persistent: boolLabel(byID.persistent.policy.persist.enabled)
      }
    }
  ];
}

export function ProfileComparisonTable({
  profiles,
  selectedProfile
}: ProfileComparisonTableProps) {
  const rows = buildRows(profiles);

  return (
    <table className="essay-table essay-profile-table">
      <thead>
        <tr>
          <th>Property</th>
          {profiles.map((profile) => (
            <th
              key={profile.id}
              className="essay-profile-table__header"
              data-selected={selectedProfile === profile.id}
            >
              {profile.id}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.label}>
            <td className="essay-table__label">{row.label}</td>
            {profiles.map((profile) => (
              <td
                key={`${row.label}-${profile.id}`}
                className="essay-profile-table__value"
                data-selected={selectedProfile === profile.id}
              >
                {row.values[profile.id]}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
