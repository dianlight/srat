"""Config flow per l'integrazione SRAT Companion."""
import logging

from homeassistant import config_entries
from homeassistant.data_entry_flow import FlowResult
from homeassistant.helpers.issue_registry import AbstractFixFlow

_LOGGER = logging.getLogger(__name__)

DOMAIN = "srat_companion"

# Importa la costante del suffisso dallo slug dall'inizializzazione
from .__init__ import ADDON_SLUG_SUFFIX

class ConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Gestisce il flusso di configurazione per SRAT Companion."""

    VERSION = 1
    CONNECTION_CLASS = config_entries.CONN_CLASS_LOCAL_POLL

    async def async_step_user(self, user_input=None) -> FlowResult:
        """Gestisce il passo iniziale quando si aggiunge manualmente l'integrazione."""
        # Applichiamo una singola istanza di questa integrazione poiché monitora un pattern.
        await self.async_set_unique_id(DOMAIN) # L'ID unico per l'integrazione principale
        self._abort_if_unique_id_configured()

        if user_input is not None:
            return self.async_create_entry(
                title=f"Monitoraggio Add-on: *{ADDON_SLUG_SUFFIX}",
                data={}, # Nessun dato utente necessario poiché lo slug è un pattern costante
            )

        # Mostra un modulo di conferma poiché non è richiesto alcun input
        return self.async_show_form(
            step_id="user",
            description_placeholders={"addon_suffix": ADDON_SLUG_SUFFIX}
        )

    async def async_step_discovery(self, discovery_info=None) -> FlowResult:
        """Gestisce il rilevamento dell'add-on in esecuzione (o installato)."""
        _LOGGER.debug("Passo di rilevamento avviato con info: %s", discovery_info)
        if discovery_info is None:
            return self.async_abort(reason="no_discovery_info")

        # Le informazioni di rilevamento conterranno l'addon_slug che è stato trovato.
        discovered_slug = discovery_info.get("addon") # La chiave 'addon' è usata nei dati di async_init in __init__.py
        if not discovered_slug or not discovered_slug.endswith(ADDON_SLUG_SUFFIX):
            _LOGGER.warning(
                "Rilevamento avviato per uno slug add-on inatteso: %s. Atteso suffisso: %s",
                discovered_slug, ADDON_SLUG_SUFFIX
            )
            return self.async_abort(reason="unexpected_addon_slug")

        # Usa lo slug dell'add-on scoperto come ID unico per l'ingresso di configurazione di scoperta.
        # Questo permette a più add-on corrispondenti al pattern di essere offerti per la configurazione.
        await self.async_set_unique_id(discovered_slug)
        self._abort_if_unique_id_configured()

        self.context["title_placeholders"] = {"name": discovered_slug}

        return self.async_show_form(
            step_id="discovery_confirm",
            description_placeholders={"addon_slug": discovered_slug},
        )

    async def async_step_discovery_confirm(self, user_input=None) -> FlowResult:
        """Conferma il rilevamento."""
        if user_input is not None:
            # Crea l'ingresso di configurazione per l'addon specifico rilevato.
            # Lo slug è derivato dall'unique_id.
            addon_slug = self.unique_id
            return self.async_create_entry(
                title=f"Rilevato Add-on: {addon_slug}",
                data={"addon": addon_slug}, # Memorizza lo slug rilevato
            )
        return self.async_abort(reason="not_confirmed")


class AddonRepairFlow(AbstractFixFlow):
    """Flusso di riparazione per l'addon non avviato."""

    # La traduzione per questo flusso è definita in strings.json sotto "issues"
    # Il fix_flow nella chiamata create_issue deve corrispondere a questo ID.
    @property
    def flow_id(self) -> str:
        """Ritorna l'ID del flusso di riparazione."""
        # Questo ID deve essere generico e il contesto specifico dell'addon sarà passato
        # dal chiamante (create_issue).
        return f"{DOMAIN}_start_addon_fix"

    async def async_step_init(self, user_input=None) -> FlowResult:
        """Gestisce il passo iniziale del flusso di riparazione."""
        # Ottieni lo slug dell'addon dal contesto dell'issue
        addon_slug = self.context.get("addon_slug")
        if not addon_slug:
            _LOGGER.error("Addon slug non trovato nel contesto del flusso di riparazione.")
            return self.async_abort(reason="missing_addon_slug")

        return self.async_show_form(
            step_id="confirm_start",
            description_placeholders={"addon_slug": addon_slug},
        )

    async def async_step_confirm_start(self, user_input=None) -> FlowResult:
        """Conferma ed esegue l'azione di avvio."""
        addon_slug = self.context.get("addon_slug")
        if not addon_slug:
            return self.async_abort(reason="missing_addon_slug")

        if user_input is not None:
            try:
                _LOGGER.info("Tentativo di avviare l'addon '%s' tramite il flusso di riparazione.", addon_slug)
                await self.hass.services.async_call(
                    "hassio", "addon_start", {"addon": addon_slug}, blocking=True
                )
                _LOGGER.info("Addon '%s' avviato con successo tramite il flusso di riparazione.", addon_slug)
                return self.async_show_success(description_placeholders={"addon_slug": addon_slug})
            except Exception as err:
                _LOGGER.error("Impossibile avviare l'addon '%s' tramite il flusso di riparazione: %s", addon_slug, err)
                return self.async_show_form(
                    step_id="confirm_start",
                    description_placeholders={"addon_slug": addon_slug, "error": str(err)},
                    errors={"base": "failed_to_start"}
                )
        return self.async_show_form(step_id="confirm_start") # Non dovrebbe accadere

