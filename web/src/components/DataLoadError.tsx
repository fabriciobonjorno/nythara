import { UiIcon } from "./UiIcon";

export function DataLoadError({ onRetry }: { onRetry: () => void }) {
  return <div className="data-load-error" role="alert">
    <span aria-hidden="true"><UiIcon name="warning" /></span>
    <div>
      <strong>Os arquivos de Nythara não responderam</strong>
      <p>Não foi possível carregar catálogo e coleção. Confira se a API local do jogo está ativa e tente reconectar.</p>
    </div>
    <button className="secondary-button" type="button" onClick={onRetry}>Tentar novamente</button>
  </div>;
}
