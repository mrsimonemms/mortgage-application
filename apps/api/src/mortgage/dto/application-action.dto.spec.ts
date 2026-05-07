import { plainToInstance } from 'class-transformer';
import { validate } from 'class-validator';

import { ApplicationActionDto } from './application-action.dto';

// These tests pin down the API contract for the new submit_property_valuation
// action: the value must be present (no API-side default), must be a number,
// and must be positive. Constraints mirror the worker's invariants so the
// workflow only ever sees positive values via the property-valuation-
// submitted signal.

describe('ApplicationActionDto: submit_property_valuation', () => {
  function buildAction(propertyValuation: unknown): ApplicationActionDto {
    return plainToInstance(ApplicationActionDto, {
      type: 'submit_property_valuation',
      propertyValuation,
    });
  }

  it('accepts a positive numeric property value', async () => {
    const dto = buildAction({ propertyValue: 350000 });
    const errors = await validate(dto);
    expect(errors).toEqual([]);
    expect(dto.propertyValuation?.propertyValue).toBe(350000);
  });

  it('rejects a missing property value (no API default is applied)', async () => {
    const dto = buildAction({});
    const errors = await validate(dto);
    expect(errors.length).toBeGreaterThan(0);
  });

  it('rejects zero', async () => {
    const dto = buildAction({ propertyValue: 0 });
    const errors = await validate(dto);
    expect(errors.length).toBeGreaterThan(0);
  });

  it('rejects a negative property value', async () => {
    const dto = buildAction({ propertyValue: -1 });
    const errors = await validate(dto);
    expect(errors.length).toBeGreaterThan(0);
  });

  it('rejects a non-numeric property value', async () => {
    const dto = buildAction({
      propertyValue: 'three hundred and fifty thousand',
    });
    const errors = await validate(dto);
    expect(errors.length).toBeGreaterThan(0);
  });

  it('rejects an entirely missing propertyValuation block', async () => {
    const dto = plainToInstance(ApplicationActionDto, {
      type: 'submit_property_valuation',
    });
    const errors = await validate(dto);
    expect(errors.length).toBeGreaterThan(0);
  });
});
