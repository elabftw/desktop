<script lang='ts'>
  type Props = { snapshot?: string; source?: string; linked: boolean };
  let {snapshot, source, linked}: Props = $props();

  function parseJSON(value: unknown): unknown {
    if (typeof value !== 'string') return value;
    try { return JSON.parse(value); } catch { return null; }
  }
  function object(value: unknown): Record<string, unknown> {
    const parsed = parseJSON(value);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
  }
  function list(value: unknown): unknown[] {
    const parsed = parseJSON(value);
    if (Array.isArray(parsed)) return parsed;
    const data = object(parsed);
    return Array.isArray(data.data) ? data.data : [];
  }
  function toText(value: unknown): string {
    if (value == null) return '';
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value);
    return JSON.stringify(value);
  }

  const data = $derived(object(snapshot));
  const tags = $derived(typeof data.tags === 'string'
    ? data.tags.split('|').filter(Boolean)
    : list(data.tags).map(tag => {
      const obj = object(tag);
      return typeof tag === 'string' ? tag : toText(obj.tag ?? obj.name);
    }).filter(Boolean));
  const metadata = $derived.by(() => {
    const fields = object(object(data.metadata).extra_fields);
    return Object.entries(fields).map(([name, field]) => {
      const obj = object(field);
      const display = obj.value !== undefined ? obj.value : field;
      return {name, value: toText(display) + (obj.units ? ` ${obj.units}` : '')};
    });
  });
  const links = $derived([
    ...list(data.itemsLinks).map(item => {
      const obj = object(item);
      return {type: 'Resource', name: toText(obj.title ?? obj.name ?? obj.entityid ?? obj.item_id ?? obj.id)};
    }),
    ...list(data.experimentsLinks).map(item => {
      const obj = object(item);
      return {type: 'Experiment', name: toText(obj.title ?? obj.name ?? obj.entityid ?? obj.experiment_id ?? obj.id)};
    }),
  ]);
  let hint = $derived(linked ? 'Pull from eLabFTW to load this information.' : 'Not linked to an eLabFTW instance.');
</script>

{#each ['Tags', 'Metadata', 'Links'] as section}
  <details class='panel'>
    <summary class='entry-panel-heading'>
      <h3>{section}{#if snapshot} ({section === 'Tags' ? tags.length : section === 'Metadata' ? metadata.length : links.length}){/if}</h3>
      <svg class='entry-chevron' aria-hidden='true' width='16' height='16' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='m6 9 6 6 6-6'/></svg>
    </summary>
    <div class='entry-detail-content'>
      {#if !snapshot}
        <p class='description'>{hint}</p>
      {:else if section === 'Tags'}
        {#if tags.length}
          <div class='entry-tag-list'>{#each tags as tag}<span class='entry-tag'>{tag}</span>{/each}</div>
        {:else}<p class='description'>No tags.</p>{/if}
      {:else if section === 'Metadata'}
        {#if metadata.length}
          {#each metadata as field}
            <div class='entry-readonly-row'><span>{field.name}</span><span>{field.value}</span></div>
          {/each}
        {:else}<p class='description'>No custom fields.</p>{/if}
      {:else}
        {#if links.length}
          {#each links as link}
            <div class='entry-readonly-row'><span>{link.name || 'Untitled'}</span><span>{link.type}</span></div>
          {/each}
        {:else}<p class='description'>No linked entries.</p>{/if}
      {/if}
      {#if snapshot && source}<p class='description'>Last pulled from {source}</p>{/if}
    </div>
  </details>
{/each}
