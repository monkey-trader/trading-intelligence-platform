import { readContractConfig, type ContractConfig } from '@/config';
import { loadOpenApi } from '@/openapi';
import { OpenApiValidator } from '@/validator';
import { request, type HttpRequest, type HttpResponse } from '@/httpClient';

/**
 * Gemeinsamer Kontext der Vertragstests. Wird beim Laden einer *.contract.test.ts-Datei gebaut:
 * fehlt CONTRACT_BASE_URL, scheitert die Datei sofort mit einer klaren Meldung.
 */
export type ContractContext = Readonly<{
  config: ContractConfig;
  validator: OpenApiValidator;
  send: (httpRequest: HttpRequest) => Promise<HttpResponse>;
}>;

export function createContractContext(): ContractContext {
  const config = readContractConfig(process.env);
  const validator = OpenApiValidator.create(loadOpenApi());
  return {
    config,
    validator,
    send: (httpRequest) => request(config.baseUrl, httpRequest),
  };
}
