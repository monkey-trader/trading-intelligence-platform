// Composition Root: einzige Stelle, die Infrastructure und Presentation verdrahtet.
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
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

createRoot(rootElement).render(
  <StrictMode>
    <OpenPositionsPage
      gateway={openPositionsGateway}
      chartGateway={positionChartGateway}
      createChartRenderer={createLightweightChartRenderer}
    />
  </StrictMode>,
);
