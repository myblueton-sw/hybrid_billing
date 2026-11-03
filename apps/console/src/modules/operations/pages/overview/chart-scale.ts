export function chartScale(maximum: bigint) {
  const value = maximum > 0n ? maximum : 1n;
  const magnitude = 10n ** BigInt(value.toString().length - 1);
  const bound = [1n, 2n, 5n, 10n]
    .map((n) => n * magnitude)
    .find((n) => n >= value)!;
  const top = bound < 2n ? 2n : bound;
  return { ceiling: top + (top / 10n || 1n), ticks: [top, top / 2n, 0n] };
}
export const chartPercent = (value: bigint, ceiling: bigint) =>
  Number((value * 10000n) / ceiling) / 100;
