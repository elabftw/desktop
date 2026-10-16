
<script lang='ts'>
  type Props = { snapshot?: string; source?: string; linked: boolean };
  let {snapshot, source, linked}: Props = $props();

  type RemoteLink = {
    title?: string;
    name?: string;
    entityid?: number;
    item_id?: number;
    experiment_id?: number;
    id?: number;
  };

  type Metadata = {
    extra_fields?: Record<string, {value?: unknown; unit?: string; units?: string | string[]}>;
  };

  type RemoteSnapshot = {
    tags?: string;
    metadata?: string | Metadata;
    itemsLinks?: string | RemoteLink[] | {data: RemoteLink[]};
    experimentsLinks?: string | RemoteLink[] | {data: RemoteLink[]};
  };

  // Parse JSON strings while allowing already decoded values
  function parseJSON<T>(value: string | T | null | undefined, fallback: T): T {
    if (value == null) return fallback;
    if (typeof value !== 'string') return value;
    try { return JSON.parse(value) ?? fallback; } catch { return fallback; }
  }
  // Extract links from arrays or responses wrapped in data
  function linkList(value: RemoteSnapshot['itemsLinks']): RemoteLink[] {
    const parsed = parseJSON<RemoteLink[] | {data: RemoteLink[]}>(value, []);
    return Array.isArray(parsed) ? parsed : Array.isArray(parsed.data) ? parsed.data : [];
  }

  // Parse the saved remote information
  const data = $derived(parseJSON<RemoteSnapshot>(snapshot, {}));
  // Split pipe-separated tags into a list
  const tags = $derived(data.tags?.split('|').filter(Boolean) ?? []);
  // Metadata is itself a JSON string in the eLabFTW API response
  const metadata = $derived(
    Object.entries(parseJSON<Metadata>(data.metadata, {}).extra_fields ?? {}).map(([name, field]) => ({
      name,
      value: String(field.value ?? '') + (field.unit ? ` ${field.unit}` : typeof field.units === 'string' ? ` ${field.units}` : ''),
    }))
  );
  // Combine linked resources and experiments
  const links = $derived([
    ...linkList(data.itemsLinks).map(item => ({
      type: 'Resource',
      name: String(item.title ?? item.name ?? item.entityid ?? item.item_id ?? item.id ?? ''),
    })),
    ...linkList(data.experimentsLinks).map(item => ({
      type: 'Experiment',
      name: String(item.title ?? item.name ?? item.entityid ?? item.experiment_id ?? item.id ?? ''),
    })),
  ]);

  const hint = $derived(linked ? 'Pull from eLabFTW to load this information.' : 'Not linked to an eLabFTW instance.');
</script>

<!-- Read-only panels for remote tags, metadata and links -->
{#each ['Tags', 'Metadata', 'Links'] as section (section)}
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
          <div class='entry-tag-list'>{#each tags as tag, i (i)}<span class='entry-tag'>{tag}</span>{/each}</div>
        {:else}<p class='description'>No tags.</p>{/if}
      {:else if section === 'Metadata'}
        {#if metadata.length}
          {#each metadata as field (field.name)}
            <div class='entry-readonly-row'><span>{field.name}</span><span>{field.value}</span></div>
          {/each}
        {:else}<p class='description'>No custom fields.</p>{/if}
      {:else}
        {#if links.length}
          {#each links as link, i (i)}
            <div class='entry-readonly-row'><span>{link.name || 'Untitled'}</span><span>{link.type}</span></div>
          {/each}
        {:else}<p class='description'>No linked entries.</p>{/if}
      {/if}
      {#if snapshot && source}<p class='description'>Last pulled from {source}</p>{/if}
    </div>
  </details>
{/each}
