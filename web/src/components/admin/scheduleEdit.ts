// How to turn one weekly schedule group into another without losing history.
//
// Each day of a weekly group is its own schedule row, and a row's completions
// are deleted with it. So an edit keeps every row it can: days kept are
// updated in place, a removed day's row is moved to an added day, and rows are
// deleted or created only for the difference in the number of days.

export interface WeeklyEditPlan {
  /** [scheduleId, dayOfWeek]: rows to update, keeping or moving their day. */
  update: Array<[number, number]>;
  /** Schedule ids to delete (days removed beyond those added). */
  remove: number[];
  /** Days to create rows for (days added beyond those removed). */
  create: number[];
}

export function planWeeklyEdit(days: number[], scheduleIds: number[], newDays: number[]): WeeklyEditPlan {
  const byDay = new Map<number, number>();
  days.forEach((d, i) => byDay.set(d, scheduleIds[i]));
  const want = [...new Set(newDays)].sort((a, b) => a - b);
  const added = want.filter(d => !byDay.has(d));
  const removed = [...byDay.keys()].filter(d => !want.includes(d)).sort((a, b) => a - b);

  const update: Array<[number, number]> = [];
  for (const d of want) {
    const id = byDay.get(d);
    if (id != null) update.push([id, d]);
  }
  removed.forEach((d, i) => {
    if (i < added.length) update.push([byDay.get(d)!, added[i]]);
  });
  return {
    update,
    remove: removed.slice(added.length).map(d => byDay.get(d)!),
    create: added.slice(removed.length),
  };
}
