<script lang="ts">
  import type { TimelineEntry } from '$lib/types';
  import { formatTime } from '$lib/utils';

  let { timeline }: { timeline: TimelineEntry[] } = $props();
</script>

<section class="card">
  <h2>Audit Timeline</h2>
  {#if timeline.length === 0}
    <p class="muted">No timeline entries yet.</p>
  {:else}
    <ol class="timeline">
      {#each [...timeline].reverse() as entry, i (i)}
        <li class="timeline-entry">
          <div class="entry-connector">
            <span class="entry-dot dot-{entry.status}"></span>
            {#if i < timeline.length - 1}
              <span class="entry-line"></span>
            {/if}
          </div>
          <div class="entry-body">
            <div class="entry-main">
              <span class="entry-step mono">{entry.step}</span>
              <span class="badge tbadge-{entry.status}">{entry.status}</span>
              <span class="entry-time">{formatTime(entry.timestamp)}</span>
            </div>
            {#if entry.details}
              <p class="entry-details">{entry.details}</p>
            {/if}
            {#if entry.metadata && Object.keys(entry.metadata).length > 0}
              <dl class="entry-meta">
                {#each Object.entries(entry.metadata) as [k, v] (k)}
                  <dt>{k}</dt>
                  <dd class="mono">{v}</dd>
                {/each}
              </dl>
            {/if}
          </div>
        </li>
      {/each}
    </ol>
  {/if}
</section>

<style>
  .timeline {
    list-style: none;
  }

  .timeline-entry {
    display: flex;
    gap: 12px;
  }

  .entry-connector {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex-shrink: 0;
    width: 14px;
  }

  .entry-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    border: 2px solid currentColor;
    flex-shrink: 0;
    margin-top: 6px;
  }

  .dot-started {
    color: #3b82f6;
    background: #eff6ff;
  }

  .dot-completed {
    color: #16a34a;
    background: #f0fdf4;
  }

  .dot-failed {
    color: #dc2626;
    background: #fef2f2;
  }

  .dot-waiting {
    color: #d97706;
    background: #fffbeb;
  }

  .entry-line {
    flex: 1;
    width: 2px;
    background: #e5e7eb;
    margin: 4px 0 0;
    min-height: 16px;
  }

  .entry-body {
    flex: 1;
    padding-bottom: 18px;
    min-width: 0;
  }

  .entry-main {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .entry-step {
    font-weight: 600;
    color: #111827;
  }

  .entry-time {
    font-size: 12px;
    color: #9ca3af;
    margin-left: auto;
  }

  .entry-details {
    font-size: 13px;
    color: #374151;
    margin-top: 5px;
    line-height: 1.4;
  }

  .entry-meta {
    display: grid;
    grid-template-columns: auto 1fr;
    column-gap: 12px;
    row-gap: 2px;
    margin-top: 8px;
    padding: 8px 10px;
    background: #f9fafb;
    border: 1px solid #f3f4f6;
    border-radius: 4px;
    font-size: 12px;
  }

  .entry-meta dt {
    color: #6b7280;
    white-space: nowrap;
  }

  .entry-meta dd {
    color: #374151;
    word-break: break-all;
  }

  /* Timeline step status badge colours */
  .tbadge-started {
    background: #eff6ff;
    color: #1e40af;
    border-color: #bfdbfe;
  }

  .tbadge-completed {
    background: #f0fdf4;
    color: #15803d;
    border-color: #bbf7d0;
  }

  .tbadge-failed {
    background: #fef2f2;
    color: #b91c1c;
    border-color: #fecaca;
  }

  .tbadge-waiting {
    background: #fffbeb;
    color: #92400e;
    border-color: #fde68a;
  }
</style>
