import type { ContractSymbolSource } from '@/domain/position/interfaces/ContractSymbolSource';
import type { BingXGetClient } from '@/infrastructure/bingx/BingXGetClient';
import { UnexpectedBingXResponseError } from '@/infrastructure/bingx/errors';

const CONTRACTS_PATH = '/openApi/swap/v2/quote/contracts';
// Das Journal deckt USDT-M-Futures ab, USDC-Kontrakte werden nicht durchsucht.
const SETTLEMENT_CURRENCY = 'USDT';

/** Liefert alle USDT-Kontrakte von BingX, auch pausierte, sortiert. */
export class BingXContractSymbolSource implements ContractSymbolSource {
  constructor(private readonly client: BingXGetClient) {}

  async listContractSymbols(): Promise<string[]> {
    const data = await this.client.get(CONTRACTS_PATH, {});
    // Eine leere Liste ist hier nie plausibel und würde die Suche still überspringen.
    if (!Array.isArray(data) || data.length === 0) {
      throw new UnexpectedBingXResponseError(
        'Antwort von quote/contracts ist keine Liste oder leer.',
      );
    }
    const symbols = data.flatMap((rawContract: unknown) => {
      if (
        typeof rawContract !== 'object' ||
        rawContract === null ||
        !('symbol' in rawContract) ||
        !('currency' in rawContract) ||
        typeof rawContract.symbol !== 'string' ||
        typeof rawContract.currency !== 'string'
      ) {
        throw new UnexpectedBingXResponseError(
          'quote/contracts enthält einen Eintrag ohne symbol oder currency.',
        );
      }
      return rawContract.currency === SETTLEMENT_CURRENCY ? [rawContract.symbol] : [];
    });
    return [...new Set(symbols)].sort();
  }
}
