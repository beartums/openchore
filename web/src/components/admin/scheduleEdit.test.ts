import { describe, it, expect } from 'vitest';
import { planWeeklyEdit } from './scheduleEdit';

describe('planWeeklyEdit', () => {
  it('updates every row in place when the days do not change', () => {
    expect(planWeeklyEdit([1, 3], [10, 11], [3, 1])).toEqual({ update: [[10, 1], [11, 3]], remove: [], create: [] });
  });

  it('moves a removed day to an added one instead of deleting it', () => {
    // Mon/Wed -> Mon/Thu: Wednesday's row becomes Thursday's.
    expect(planWeeklyEdit([1, 3], [10, 11], [1, 4])).toEqual({ update: [[10, 1], [11, 4]], remove: [], create: [] });
  });

  it('creates only the days beyond those removed', () => {
    expect(planWeeklyEdit([1], [10], [2, 5])).toEqual({ update: [[10, 2]], remove: [], create: [5] });
  });

  it('deletes only the days beyond those added', () => {
    expect(planWeeklyEdit([1, 3, 5], [10, 11, 12], [3])).toEqual({ update: [[11, 3]], remove: [10, 12], create: [] });
  });

  it('ignores duplicate days', () => {
    expect(planWeeklyEdit([0], [7], [6, 6])).toEqual({ update: [[7, 6]], remove: [], create: [] });
  });
});
