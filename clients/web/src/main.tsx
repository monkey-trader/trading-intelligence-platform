// Composition Root: einzige Stelle, die Infrastructure und Presentation verdrahtet.
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createHttpAccountSummaryGateway } from '@/infrastructure/api/httpAccountSummaryGateway';
import { createHttpAssetIconsGateway } from '@/infrastructure/api/httpAssetIconsGateway';
import { AppHeader } from '@/presentation/shared/components/AppHeader';
import { createHttpInitialStopLossGateway } from '@/infrastructure/api/httpInitialStopLossGateway';
import { createHttpOpenPositionsGateway } from '@/infrastructure/api/httpOpenPositionsGateway';
import { createHttpPositionChartGateway } from '@/infrastructure/api/httpPositionChartGateway';
import { createLightweightChartRenderer } from '@/infrastructure/chart/lightweightChartRenderer';
import { OpenPositionsPage } from '@/presentation/position/components/OpenPositionsPage';
import '@/styles/design-tokens.css';
import '@/styles/global.css';

const rootElement = document.getElementById('root');
if (!rootElement) throw new Error('Element #root fehlt in index.html');

const openPositionsGateway = createHttpOpenPositionsGateway(
  fetch,
  import.meta.env.VITE_API_BASE ?? '/api',
);

const apiBase = import.meta.env.VITE_API_BASE ?? '/api';
const positionChartGateway = createHttpPositionChartGateway(fetch, apiBase);
const assetIconsGateway = createHttpAssetIconsGateway(fetch, apiBase);

createRoot(rootElement).render(
  <StrictMode>
    <AppHeader />
    <OpenPositionsPage
      accountSummaryGateway={createHttpAccountSummaryGateway(fetch, apiBase)}
      assetIconsGateway={assetIconsGateway}
      gateway={openPositionsGateway}
      initialStopLossGateway={createHttpInitialStopLossGateway()}
      chartGateway={positionChartGateway}
      createChartRenderer={createLightweightChartRenderer}
    />
  </StrictMode>,
);
