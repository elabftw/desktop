<script lang='ts'>
  type Props = {snapshot?: string; emptyLabel?: string};
  let {snapshot, emptyLabel = ''}: Props = $props();

  function fromSnapshot(value?: string): {kind: string; title: string; color: string}[] {
    if (!value) return [];
    try {
      const data = JSON.parse(value) as Record<string, unknown>;
      const safeColor = (color: unknown): string => {
        const hex = typeof color === 'string' ? color.replace(/^#/, '') : '';
        return /^[0-9a-f]{6}$/i.test(hex) ? `#${hex}` : 'var(--light-gray)';
      };
      const badges = [
        {kind: 'Category', title: data.categoryTitle, color: data.categoryColor},
        {kind: 'Status', title: data.statusTitle, color: data.statusColor},
      ];
      return badges.filter(badge => typeof badge.title === 'string' && badge.title.trim() !== '')
        .map(badge => ({kind: badge.kind, title: String(badge.title), color: safeColor(badge.color)}));
    } catch {
      return [];
    }
  }

  let labels = $derived(fromSnapshot(snapshot));
</script>

{#if labels.length}
  <span class='flex gap-1'>
    {#each labels as badge (badge.kind)}
      <span title={badge.kind}>
        <span class='catstat-dot' style:background-color={badge.color} aria-hidden='true'></span>
        {badge.title}
      </span>
    {/each}
  </span>
{:else if emptyLabel}
  <span class='description'>{emptyLabel}</span>
{/if}
