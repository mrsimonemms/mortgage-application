<script lang="ts">
  import * as api from '$lib/api';
  import type { MortgageApplication } from '$lib/types';
  import { statusLabel } from '$lib/utils';

  let {
    app,
    isTerminal,
    isCreditCheckPending,
    onRefresh,
  }: {
    app: MortgageApplication;
    isTerminal: boolean;
    isCreditCheckPending: boolean;
    onRefresh: () => Promise<void>;
  } = $props();

  let creditResult: 'approved' | 'rejected' = $state('approved');
  let creditRef = $state('');
  let creditError = $state('');
  let creditLoading = $state(false);

  async function handleCreditCheck(e: SubmitEvent) {
    e.preventDefault();
    creditLoading = true;
    creditError = '';
    try {
      await api.submitCreditCheck(app.applicationId, {
        result: creditResult,
        reference: creditRef.trim() || undefined,
      });
      creditRef = '';
      await onRefresh();
    } catch (err) {
      creditError =
        err instanceof Error ? err.message : 'Failed to submit credit check';
    } finally {
      creditLoading = false;
    }
  }
</script>

<section class="card">
  <h2>Available Actions</h2>
  {#if isTerminal}
    <p class="muted">
      No actions available. The workflow is in a terminal state:
      <strong>{statusLabel(app.status)}</strong>.
    </p>
  {:else if isCreditCheckPending}
    <div class="action-block">
      <h3>Submit Credit Check Result</h3>
      <p class="hint">
        The workflow is waiting for an external credit check signal. Submit the
        outcome below to unblock it.
      </p>
      <form onsubmit={handleCreditCheck}>
        <div class="field">
          <label for="credit-result">Result</label>
          <select id="credit-result" bind:value={creditResult}>
            <option value="approved">Approved</option>
            <option value="rejected">Rejected</option>
          </select>
        </div>
        <div class="field">
          <label for="credit-ref">Reference (optional)</label>
          <input
            id="credit-ref"
            type="text"
            bind:value={creditRef}
            placeholder="e.g. CB-12345"
            class="mono"
          />
        </div>
        {#if creditError}
          <p class="error">{creditError}</p>
        {/if}
        <button type="submit" class="btn-primary" disabled={creditLoading}>
          {creditLoading ? 'Submitting…' : 'Submit Credit Check'}
        </button>
      </form>
    </div>
  {:else}
    <p class="muted">
      No external input required at this step. The workflow is progressing
      automatically via Temporal.
    </p>
  {/if}
</section>

<style>
  .action-block {
    border: 1px solid #fde68a;
    background: #fffbeb;
    border-radius: 6px;
    padding: 14px;
  }

  .action-block h3 {
    font-size: 13px;
    font-weight: 600;
    color: #92400e;
    margin-bottom: 6px;
  }

  .action-block .hint {
    margin-bottom: 12px;
  }
</style>
