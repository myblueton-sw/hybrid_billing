export const addressCursor = (value: string | null) => (value !== null && value.length <= 4096 ? value : "");
export const addressSize = (value: string | null, fallback: number) => (value !== null && /^[1-9][0-9]{0,2}$/.test(value) && Number(value) <= 100 ? Number(value) : fallback);
