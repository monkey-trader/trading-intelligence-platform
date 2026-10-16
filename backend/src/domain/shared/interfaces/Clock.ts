/** Port für die aktuelle Zeit in Epoch-Millisekunden, damit Tests eine feste Zeit injizieren können. */
export type Clock = {
  now(): number;
};
