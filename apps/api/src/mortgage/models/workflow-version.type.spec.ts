import { deriveWorkflowVersion } from './workflow-version.type';

describe('deriveWorkflowVersion', () => {
  it('maps a v1 build ID to v1', () => {
    expect(deriveWorkflowVersion('mortgage-worker-v1')).toBe('v1');
  });

  it('maps a v2 build ID to v2', () => {
    expect(deriveWorkflowVersion('mortgage-worker-v2')).toBe('v2');
  });

  it('returns unknown for an undefined build ID', () => {
    expect(deriveWorkflowVersion(undefined)).toBe('unknown');
  });

  it('returns unknown for null', () => {
    expect(deriveWorkflowVersion(null)).toBe('unknown');
  });

  it('returns unknown for an empty string', () => {
    expect(deriveWorkflowVersion('')).toBe('unknown');
  });

  it('returns unknown for an unrecognised version suffix', () => {
    // v3 is not (yet) a known version in this PoC; collapse to unknown so
    // the UI never has to handle a value it does not recognise.
    expect(deriveWorkflowVersion('mortgage-worker-v3')).toBe('unknown');
  });

  it('returns unknown for a build ID with no version suffix', () => {
    expect(deriveWorkflowVersion('mortgage-worker')).toBe('unknown');
  });

  it('matches against the trailing version segment regardless of deployment prefix', () => {
    // Treat the prefix as opaque so renaming the deployment does not break
    // the mapping.
    expect(deriveWorkflowVersion('some-other-prefix-v1')).toBe('v1');
    expect(deriveWorkflowVersion('some-other-prefix-v2')).toBe('v2');
  });
});
