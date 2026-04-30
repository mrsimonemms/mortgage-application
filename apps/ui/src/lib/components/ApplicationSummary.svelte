<script lang="ts">
  import type { MortgageApplication } from '$lib/types';
  import { formatTime, statusLabel } from '$lib/utils';

  let {
    app,
    isTerminal,
    refreshing,
    refreshError,
    onRefresh,
    refreshTimeout,
  }: {
    app: MortgageApplication;
    isTerminal: boolean;
    refreshing: boolean;
    refreshError: string;
    onRefresh: () => void;
    refreshTimeout: number;
  } = $props();
</script>

<section class="card">
  <div class="card-header">
    <h2>Application Summary</h2>
    <button class="btn-secondary" onclick={onRefresh} disabled={refreshing}>
      {refreshing ? 'Refreshing…' : 'Refresh'}
    </button>
  </div>
  {#if refreshError}
    <p class="error">{refreshError}</p>
  {/if}
  <dl class="summary">
    <dt>Application ID</dt>
    <dd class="mono">{app.applicationId}</dd>
    <dt>Applicant</dt>
    <dd>{app.applicantName}</dd>
    <dt>Status</dt>
    <dd>
      <span class="badge status-{app.status}">
        {statusLabel(app.status)}
      </span>
    </dd>
    <dt>Current Step</dt>
    <dd class="mono">{app.currentStep}</dd>
    {#if app.offerId}
      <dt>Offer ID</dt>
      <dd class="mono">{app.offerId}</dd>
    {/if}
    <dt>Created</dt>
    <dd>{formatTime(app.createdAt)}</dd>
    <dt>Updated</dt>
    <dd>{formatTime(app.updatedAt)}</dd>
  </dl>
  {#if !isTerminal}
    <p class="status-note">
      Auto-refreshing every {Math.round(refreshTimeout / 1000)}s.
    </p>
  {:else}
    <p class="status-note terminal">Workflow has reached a terminal state.</p>
  {/if}
</section>

<style>
  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
    padding-bottom: 10px;
    border-bottom: 1px solid #f3f4f6;
  }

  /* Override card h2 default when inside the header row */
  .card-header h2 {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }

  .summary {
    display: grid;
    grid-template-columns: auto 1fr;
    column-gap: 16px;
    row-gap: 2px;
    font-size: 13px;
  }

  .summary dt {
    color: #6b7280;
    font-weight: 500;
    white-space: nowrap;
    padding: 3px 0;
  }

  .summary dd {
    color: #111827;
    padding: 3px 0;
    word-break: break-all;
  }

  .status-note {
    font-size: 12px;
    color: #9ca3af;
    margin-top: 12px;
  }

  .status-note.terminal {
    color: #16a34a;
  }

  /* Application status badge colours */
  .status-submitted {
    background: #f3f4f6;
    color: #374151;
    border-color: #d1d5db;
  }

  .status-credit_check_pending {
    background: #fffbeb;
    color: #92400e;
    border-color: #fde68a;
  }

  .status-offer_reserved {
    background: #eff6ff;
    color: #1e40af;
    border-color: #bfdbfe;
  }

  .status-completed {
    background: #f0fdf4;
    color: #15803d;
    border-color: #bbf7d0;
  }

  .status-rejected {
    background: #fef2f2;
    color: #b91c1c;
    border-color: #fecaca;
  }

  .status-compensation_required {
    background: #fff7ed;
    color: #c2410c;
    border-color: #fed7aa;
  }

  .status-compensated {
    background: #fdf4ff;
    color: #7e22ce;
    border-color: #e9d5ff;
  }
</style>
