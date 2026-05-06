<script lang="ts">
  import type { MortgageApplication } from '$lib/types';
  import {
    formatTime,
    isNonRunningTerminal,
    lifecycleLabel,
    statusLabel,
  } from '$lib/utils';

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

  let now = $state(Date.now());

  // Workflows in TERMINATED, CANCELLED, TIMED_OUT or FAILED have stopped
  // running without giving the workflow a chance to clear transient SLA
  // fields, so a query response can still carry pendingDependency /
  // slaDeadline from the moment of the stop. Treat all such lifecycle
  // states as "stopped": no live SLA timer, no progress bar, and a neutral
  // per-status badge that does not imply a business outcome.
  //
  // Live SLA logic only fires when the workflow is RUNNING and a
  // dependency is pending. COMPLETED is handled by the existing
  // post-completion path (persisted slaStatus/slaBreached). Undefined
  // workflowStatus (older payloads or pre-fetch state) is treated as
  // running so behaviour is unchanged.
  const stoppedStatus = $derived(
    isNonRunningTerminal(app.workflowStatus) ? app.workflowStatus : undefined,
  );

  // Tick once a second so the elapsed-waiting label updates between API
  // refreshes. Only runs while a dependency is pending, the business state
  // is non-terminal, and the workflow is still actually running.
  $effect(() => {
    if (isTerminal || stoppedStatus !== undefined || !app.pendingSince) return;
    const id = setInterval(() => (now = Date.now()), 1000);
    return () => clearInterval(id);
  });

  // While the workflow is waiting (pendingDependency is set) the SLA row uses
  // live client-side computation against the deadline. Once the wait resolves
  // the workflow persists slaStatus / slaBreached and the row switches to
  // showing the durable outcome without a progress bar or ticking timer.
  // A stopped workflow is never considered "waiting" even if its last query
  // snapshot still carries a pendingDependency.
  const isWaiting = $derived(
    !!app.pendingDependency && stoppedStatus === undefined,
  );

  const pendingSinceMs = $derived(
    app.pendingSince ? new Date(app.pendingSince).getTime() : 0,
  );
  const slaDeadlineMs = $derived(
    app.slaDeadline ? new Date(app.slaDeadline).getTime() : 0,
  );
  // Prefer the client-side comparison while waiting so the badge transitions
  // live without needing a fresh API refresh; once finalised, trust the
  // workflow's stored value.
  const slaBreached = $derived(
    isWaiting && app.slaDeadline
      ? now > slaDeadlineMs
      : (app.slaBreached ?? false),
  );
  // Reference time for SLA duration. While waiting we use live `now` so the
  // figure ticks once a second; once the wait has resolved we freeze the
  // reference to `app.updatedAt` so the displayed duration is stable and does
  // not drift as wall-clock time keeps advancing.
  const slaReferenceTimeMs = $derived(
    isWaiting ? now : new Date(app.updatedAt).getTime(),
  );
  // Signed offset from the deadline. Negative while there is still time
  // remaining on the SLA, positive once the deadline has passed. The badge
  // already carries the outcome so this string only conveys timing context.
  const slaOffsetMs = $derived(slaReferenceTimeMs - slaDeadlineMs);
  const slaSignedDuration = $derived(
    slaOffsetMs > 0
      ? `+${formatDuration(slaOffsetMs)}`
      : `−${formatDuration(-slaOffsetMs)}`,
  );
  const slaProgress = $derived.by(() => {
    const total = slaDeadlineMs - pendingSinceMs;
    if (total <= 0) return slaBreached ? 1 : 0;
    return Math.min(1, Math.max(0, (now - pendingSinceMs) / total));
  });
  // Amber warning once more than half of the SLA window has elapsed but the
  // deadline has not yet been reached. Only meaningful while waiting; once
  // finalised the outcome is binary (met or breached).
  const slaWarning = $derived(isWaiting && !slaBreached && slaProgress > 0.5);
  // Stopped takes precedence over breach/warn/ok so a non-running terminal
  // workflow never displays a live business-outcome badge. Lifecycle states
  // are presented as their own distinct badge value.
  const slaState = $derived(
    stoppedStatus !== undefined
      ? 'stopped'
      : slaBreached
        ? 'breached'
        : slaWarning
          ? 'warn'
          : 'ok',
  );
  const slaBadgeLabel = $derived(
    stoppedStatus !== undefined
      ? lifecycleLabel(stoppedStatus)
      : slaBreached
        ? 'SLA breached'
        : slaWarning
          ? 'SLA at risk'
          : 'Within SLA',
  );
  // Show the SLA row whenever there is a deadline to display (in flight or
  // persisted), or when the workflow has been stopped while waiting on a
  // dependency that had an SLA recorded.
  const showSlaRow = $derived(
    !!app.slaDeadline &&
      (isWaiting || !!app.slaStatus || stoppedStatus !== undefined),
  );
  const dependencyLabel = $derived(
    app.pendingDependency ? humanise(app.pendingDependency) : '',
  );

  function humanise(value: string): string {
    return value.replace(/_/g, ' ');
  }

  function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    if (totalSeconds < 60) return `${totalSeconds}s`;
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}m ${seconds}s`;
  }

  function formatShortTime(iso: string): string {
    return new Date(iso).toLocaleTimeString(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  }
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
    {#if isWaiting}
      <dt>Waiting For</dt>
      <dd>
        <span class="waiting-label">{dependencyLabel}</span>
      </dd>
    {/if}
    {#if showSlaRow && app.slaDeadline}
      <dt>SLA</dt>
      <dd>
        <span
          class="badge sla-badge"
          class:sla-ok={slaState === 'ok'}
          class:sla-warn={slaState === 'warn'}
          class:sla-breached={slaState === 'breached'}
          class:sla-stopped={slaState === 'stopped'}
        >
          {slaBadgeLabel}
        </span>
      </dd>
      <dt>Target</dt>
      <dd>
        <div class="mono">
          {formatShortTime(app.slaDeadline)} ({slaSignedDuration})
        </div>
        {#if isWaiting}
          <div
            class="sla-progress"
            class:sla-progress-warn={slaState === 'warn'}
            class:sla-progress-breached={slaState === 'breached'}
            role="progressbar"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={Math.round(slaProgress * 100)}
          >
            <div
              class="sla-progress-bar"
              style="width: {slaProgress * 100}%"
            ></div>
          </div>
        {/if}
      </dd>
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

  /* SLA visibility for pending async dependencies */
  .waiting-label {
    margin-right: 8px;
    text-transform: capitalize;
  }

  .sla-badge {
    font-size: 11px;
  }

  .sla-ok {
    background: #f0fdf4;
    color: #15803d;
    border-color: #bbf7d0;
  }

  .sla-warn {
    background: #fffbeb;
    color: #b45309;
    border-color: #fde68a;
  }

  .sla-breached {
    background: #fef2f2;
    color: #b91c1c;
    border-color: #fecaca;
  }

  /* Neutral grey for externally stopped workflows. Distinct from the
     business-outcome SLA states so terminated executions are not mistaken
     for a met or breached SLA. */
  .sla-stopped {
    background: #f3f4f6;
    color: #4b5563;
    border-color: #d1d5db;
  }

  .sla-progress {
    margin-top: 4px;
    width: 100%;
    height: 4px;
    background: #f3f4f6;
    border-radius: 2px;
    overflow: hidden;
  }

  .sla-progress-bar {
    height: 100%;
    background: #16a34a;
    transition:
      width 0.5s linear,
      background-color 0.2s linear;
  }

  .sla-progress-warn .sla-progress-bar {
    background: #d97706;
  }

  .sla-progress-breached .sla-progress-bar {
    background: #dc2626;
    width: 100% !important;
  }
</style>
