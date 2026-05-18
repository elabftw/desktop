<script lang='ts'>
  import type { main } from '../../../wailsjs/go/models';

  type Props = {
    profiles: main.ProfileEntry[];
    activeProfile: string | null;
    openAddProfile: () => void;
    selectProfile: (uuid: string, name: string) => void;
  };

  let {profiles, activeProfile, openAddProfile, selectProfile}: Props = $props();

  function initials(profile: main.ProfileEntry): string {
    return profile.display_name?.trim().slice(0, 2).toUpperCase();
  }
</script>

<section class='profile-page'>
  <header class='profile-hero'>
    <div>
      <p class='eyebrow'>Welcome back</p>
      <h1>Select a profile</h1>
      <p class='hero-subtitle'>
        Choose a profile to unlock your saved entries.
      </p>
    </div>

    <button class='btn btn-primary btn-create-profile' onclick={openAddProfile}>
      <span aria-hidden='true'>+</span>
      Add profile
    </button>
  </header>

  <section class='profile-panel' aria-labelledby='profiles-title'>
    <div class='panel-header'>
      <div class='panel-title-block'>
        <div class='panel-icon' aria-hidden='true'>●</div>

        <div>
          <h2 id='profiles-title'>Profiles</h2>
          <p>
            {profiles.length === 1
              ? '1 profile available'
              : `${profiles.length} profiles available`}
          </p>
        </div>
      </div>
    </div>

    {#if profiles.length > 0}
      <div class='profiles'>
        {#each profiles as profile (profile.uuid)}
          <button
            type='button'
            class='profile-box'
            class:active={activeProfile === profile.uuid}
            class:masked={activeProfile !== null && activeProfile !== profile.uuid}
            onclick={() => selectProfile(profile.uuid, profile.display_name)}
            aria-pressed={activeProfile === profile.uuid}
          >
            <span class='profile-avatar'>{initials(profile)}</span>

            <span class='profile-content'>
              <span class='profile-name'>{profile.display_name?.trim() || profile.uuid}</span>
              <span class='profile-meta'>
                {activeProfile === profile.uuid ? 'Selected' : 'Click to unlock'}
              </span>
            </span>

            <span class='profile-action' aria-hidden='true'>
              {activeProfile === profile.uuid ? 'Unlock ↓' : 'Select →'}
            </span>
          </button>
        {/each}
      </div>
    {:else}
      <div class='empty-state'>
        <div class='empty-icon' aria-hidden='true'>+</div>
        <h3>No profiles yet</h3>
        <p>Create a profile to start saving entries.</p>

        <button class='btn btn-primary' onclick={openAddProfile}>
          Create your first profile
        </button>
      </div>
    {/if}
  </section>
</section>
