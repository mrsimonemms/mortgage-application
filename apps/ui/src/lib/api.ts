import type {
  ApplicationListItem,
  MortgageApplication,
  ScenarioOption,
} from './types';

const BASE = '/api';

async function request<T>(
  path: string,
  init?: RequestInit,
  fetchFn?: typeof fetch,
): Promise<T> {
  const fn = fetchFn ?? fetch;
  const headers = new Headers(init?.headers);
  if (init?.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  const res = await fn(`${BASE}${path}`, { ...init, headers });

  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body?.message) {
        message = Array.isArray(body.message)
          ? body.message.join(', ')
          : String(body.message);
      }
    } catch {
      // ignore parse failure
    }
    throw new Error(message);
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function getApplications(
  fetchFn?: typeof fetch,
): Promise<ApplicationListItem[]> {
  return request<ApplicationListItem[]>('/v1/applications', undefined, fetchFn);
}

export async function getScenarios(
  fetchFn?: typeof fetch,
): Promise<ScenarioOption[]> {
  const { scenarios } = await request<{ scenarios: ScenarioOption[] }>(
    '/v1/applications/scenarios',
    undefined,
    fetchFn,
  );

  return scenarios;
}

export interface StartApplicationPayload {
  applicationId: string;
  applicantName: string;
  scenario: string;
}

export async function startApplication(
  payload: StartApplicationPayload,
): Promise<void> {
  await request<void>('/v1/applications', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function getApplication(
  applicationId: string,
  fetchFn?: typeof fetch,
): Promise<MortgageApplication> {
  return request<MortgageApplication>(
    `/v1/applications/${encodeURIComponent(applicationId)}`,
    undefined,
    fetchFn,
  );
}

export interface CreditCheckPayload {
  result: 'approved' | 'rejected';
  reference?: string;
}

export async function submitCreditCheck(
  applicationId: string,
  payload: CreditCheckPayload,
): Promise<void> {
  await request<void>(
    `/v1/applications/${encodeURIComponent(applicationId)}/credit-check`,
    { method: 'POST', body: JSON.stringify(payload) },
  );
}

export async function retryCreditCheck(applicationId: string): Promise<void> {
  await request<void>(
    `/v1/applications/${encodeURIComponent(applicationId)}/retry-credit-check`,
    { method: 'POST' },
  );
}

export async function rerunApplication(
  applicationId: string,
): Promise<{ applicationId: string; workflowId: string }> {
  return request<{ applicationId: string; workflowId: string }>(
    `/v1/applications/${encodeURIComponent(applicationId)}/rerun`,
    { method: 'POST' },
  );
}
