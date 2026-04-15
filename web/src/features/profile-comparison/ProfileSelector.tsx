import { Button } from "@/components/primitives";
import type { ProfileName, ProfileSpec } from "@/features/meet-session/types";

type ProfileSelectorProps = {
  profiles: ProfileSpec[];
  selectedProfile: ProfileName;
  onSelect: (profile: ProfileName) => void;
};

export function ProfileSelector({
  profiles,
  selectedProfile,
  onSelect
}: ProfileSelectorProps) {
  return (
    <div className="essay-profile-selector" role="tablist" aria-label="REPL profiles">
      {profiles.map((profile) => (
        <Button
          key={profile.id}
          type="button"
          variant="secondary"
          className="essay-profile-selector__button"
          data-selected={selectedProfile === profile.id}
          aria-pressed={selectedProfile === profile.id}
          onClick={() => onSelect(profile.id)}
        >
          {profile.id}
        </Button>
      ))}
    </div>
  );
}
