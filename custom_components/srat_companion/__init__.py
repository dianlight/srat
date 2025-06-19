"""L'integrazione SRAT Companion."""
from __future__ import annotations

import logging
from datetime import timedelta
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.event import async_track_time_interval
from homeassistant.helpers.typing import ConfigType
from homeassistant.exceptions import ConfigEntryNotReady
from homeassistant.loader import Integration
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed
from homeassistant.helpers.issue_registry import IssueRegistry, IssueSeverity, IssueCategory, create_issue, delete_issue, async_get_issue_registry
from homeassistant.util import utcnow

_LOGGER = logging.getLogger(__name__)

DOMAIN = "srat_companion"
PLATFORMS: list[str] = [] # Questo componente non crea entità, solo rilevamento e riparazione.

# --- CONFIGURA QUESTO SUFFISSO PER GLI SLUG DEGLI ADD-ON CHE VUOI MONITORARE ---
ADDON_SLUG_SUFFIX = "_sambanas2"
# -----------------------------------------------------------------------

# Il nome dell'evento per le risposte delle informazioni degli add-on del Supervisor
EVENT_SUPERVISOR_ADDON_INFO = "hassio_addon_info"

SCAN_INTERVAL = timedelta(minutes=1) # Ogni quanto controllare lo stato degli add-on

# Questo set memorizzerà tutti gli slug degli add-on che corrispondono al pattern e che sono stati "rilevati"
# per evitare di attivare più volte il flusso di scoperta.
_DISCOVERED_ADDONS: set[str] = set()

# Questo dizionario traccia i timestamp per gli add-on che non sono in stato "started"
# Chiave: addon_slug, Valore: datetime del momento in cui è stato notato per la prima volta come non avviato
_NOT_STARTED_TIMESTAMPS: dict[str, Any] = {}


async def async_setup(hass: HomeAssistant, config: ConfigType) -> bool:
    """Imposta SRAT Companion da configuration.yaml (o per il rilevamento automatico iniziale)."""
    hass.data.setdefault(DOMAIN, {})

    # Controlla se un ingresso di configurazione per l'integrazione principale è già impostato.
    existing_entries = hass.config_entries.async_entries(DOMAIN)
    is_already_configured = any(entry.unique_id == DOMAIN for entry in existing_entries)

    if not is_already_configured:
        _LOGGER.debug("Controllo per il rilevamento automatico iniziale degli add-on con suffisso: %s", ADDON_SLUG_SUFFIX)
        try:
            hassio_integration: Integration | None = hass.data.get("hassio")
            if not hassio_integration:
                _LOGGER.warning(
                    "L'integrazione Home Assistant Supervisor (hassio) non è stata caricata durante l'avvio. "
                    "Il rilevamento automatico potrebbe essere ritardato o fallire."
                )
                return True

            # Chiama il servizio hassio per ottenere le informazioni su TUTTI gli add-on
            response: dict[str, Any] = await hass.services.async_call(
                "hassio", "addons_info", blocking=True, return_response=True
            )
            
            addons_info: dict[str, Any] = response.get("info", {})
            
            for addon_slug, addon_data in addons_info.items():
                if addon_slug.endswith(ADDON_SLUG_SUFFIX):
                    if addon_data.get("installed", False) and addon_slug not in _DISCOVERED_ADDONS:
                        _LOGGER.info(
                            "Add-on '%s' (che termina con '%s') trovato installato all'avvio. Attivazione del rilevamento automatico.",
                            addon_slug, ADDON_SLUG_SUFFIX
                        )
                        hass.async_create_task(
                            hass.config_entries.flow.async_init(
                                DOMAIN,
                                context={"source": config_entries.SOURCE_DISCOVERY},
                                data={"addon": addon_slug},
                            )
                        )
                        _DISCOVERED_ADDONS.add(addon_slug)
        except Exception as err:
            _LOGGER.error(
                "Errore durante il rilevamento automatico iniziale per add-on con suffisso '%s': %s",
                ADDON_SLUG_SUFFIX, err
            )
    else:
        _LOGGER.debug("SRAT Companion è già configurato.")

    return True


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Imposta SRAT Companion da un ingresso di configurazione."""
    _LOGGER.debug("Impostazione dell'ingresso di configurazione per SRAT Companion: %s", entry.unique_id)

    hassio_integration: Integration | None = hass.data.get("hassio")
    if not hassio_integration:
        _LOGGER.error(
            "L'integrazione Home Assistant Supervisor (hassio) non è stata caricata. "
            "SRAT Companion richiede hassio."
        )
        raise ConfigEntryNotReady("L'integrazione Hass.io non è pronta.")

    # Il coordinatore aggiornerà le informazioni su TUTTI gli add-on
    async def async_update_data():
        """Recupera i dati dal Supervisor per tutti gli add-on."""
        _LOGGER.debug("Controllo stato per tutti gli add-on Supervisor.")
        try:
            # Chiama il servizio hassio per ottenere le informazioni su TUTTI gli add-on
            await hass.services.async_call(
                "hassio", "addons_info", blocking=False
            )
            return True
        except Exception as err:
            _LOGGER.error("Errore nella chiamata hassio.addons_info: %s", err)
            raise UpdateFailed(f"Errore nella chiamata hassio.addons_info: {err}") from err

    coordinator = DataUpdateCoordinator(
        hass,
        _LOGGER,
        name="Stato Add-on Supervisor",
        update_method=async_update_data,
        update_interval=SCAN_INTERVAL,
    )
    hass.data[DOMAIN][entry.entry_id] = {"coordinator": coordinator} # Memorizza il coordinatore

    @callback
    def _async_hassio_addons_info_listener(event):
        """Ascolta gli eventi hassio_addon_info (che include le info per tutti gli add-on)."""
        all_addons_info: dict[str, Any] = event.data.get("info", {})
        issue_registry: IssueRegistry = async_get_issue_registry(hass)

        for addon_slug, addon_info in all_addons_info.items():
            if addon_slug.endswith(ADDON_SLUG_SUFFIX):
                current_state = addon_info.get("state")
                is_installed = addon_info.get("installed", False)
                
                # --- Logica di rilevamento ---
                if is_installed and addon_slug not in _DISCOVERED_ADDONS:
                    _LOGGER.info(
                        "Add-on '%s' (che termina con '%s') è installato. Attivazione del rilevamento.",
                        addon_slug, ADDON_SLUG_SUFFIX
                    )
                    hass.async_create_task(
                        hass.config_entries.flow.async_init(
                            DOMAIN,
                            context={"source": config_entries.SOURCE_DISCOVERY},
                            data={"addon": addon_slug},
                        )
                    )
                    _DISCOVERED_ADDONS.add(addon_slug)

                # --- Logica di Riparazione ---
                # ID dell'issue di riparazione specifico per questo addon
                repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
                
                if is_installed and current_state != "started":
                    if addon_slug not in _NOT_STARTED_TIMESTAMPS:
                        # Prima volta che lo vediamo non avviato, registra il timestamp
                        _NOT_STARTED_TIMESTAMPS[addon_slug] = utcnow()
                        _LOGGER.debug(
                            "Add-on '%s' è installato ma non avviato. Tracciamento del tempo.",
                            addon_slug
                        )
                    elif (utcnow() - _NOT_STARTED_TIMESTAMPS[addon_slug]) > timedelta(minutes=5):
                        # Più di 5 minuti, crea/aggiorna l'issue di riparazione
                        _LOGGER.warning(
                            "Add-on '%s' è installato ma non avviato da più di 5 minuti. Creazione issue di riparazione.",
                            addon_slug
                        )
                        create_issue(
                            hass,
                            DOMAIN,
                            repair_issue_id, # ID dell'issue dinamico
                            issue_domain=DOMAIN,
                            is_fixable=True,
                            is_persistent=True,
                            learn_more_url="https://www.home-assistant.io/integrations/hassio/", # Esempio di URL
                            severity=IssueSeverity.WARNING,
                            translation_key="addon_not_started",
                            translation_placeholders={
                                "addon_slug": addon_slug,
                                "time_threshold": "5 minuti"
                            },
                            fix_flow=f"{DOMAIN}_start_addon_fix", # Riferisce al nome del flusso di riparazione in config_flow.py
                            context={"addon_slug": addon_slug} # Passa lo slug al contesto del fix_flow
                        )
                elif current_state == "started":
                    # L'add-on è avviato, pulisci il timestamp e elimina l'issue di riparazione se esiste
                    if addon_slug in _NOT_STARTED_TIMESTAMPS:
                        del _NOT_STARTED_TIMESTAMPS[addon_slug]
                        _LOGGER.debug(
                            "Add-on '%s' è ora avviato. Pulizia timestamp not_started_since.",
                            addon_slug
                        )
                    if issue_registry.get_issue(DOMAIN, repair_issue_id):
                        _LOGGER.info(
                            "Add-on '%s' è ora avviato. Eliminazione issue di riparazione.",
                            addon_slug
                        )
                        delete_issue(hass, DOMAIN, repair_issue_id)
                # Se non è installato e non era stato rilevato, assicurati che non ci siano issue residue
                elif not is_installed:
                    if addon_slug in _DISCOVERED_ADDONS:
                        _DISCOVERED_ADDONS.remove(addon_slug)
                        _LOGGER.debug("Add-on '%s' non è più installato. Rimosso dalla cache di rilevamento.", addon_slug)
                    if addon_slug in _NOT_STARTED_TIMESTAMPS:
                        del _NOT_STARTED_TIMESTAMPS[addon_slug]
                        _LOGGER.debug("Add-on '%s' non è più installato. Rimosso dal tracciamento dei timestamp.", addon_slug)
                    if issue_registry.get_issue(DOMAIN, repair_issue_id):
                        delete_issue(hass, DOMAIN, repair_issue_id)
                        _LOGGER.info("Add-on '%s' non è più installato. Eliminazione issue di riparazione.", addon_slug)


    # Ascolta l'evento hassio_addon_info
    # Nota: il servizio hassio.addons_info pubblica un evento hassio_addon_info
    # contenente le informazioni per TUTTI gli add-on.
    entry.async_on_unload(
        hass.bus.async_listen(EVENT_SUPERVISOR_ADDON_INFO, _async_hassio_addons_info_listener)
    )

    # Attiva immediatamente il primo aggiornamento per ottenere lo stato iniziale di tutti gli add-on
    await coordinator.async_config_entry_first_refresh()

    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Scarica un ingresso di configurazione."""
    _LOGGER.debug("Scarico dell'ingresso di configurazione per SRAT Companion: %s", entry.entry_id)
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        # Assicurati di rimuovere il coordinatore dalla memoria
        coordinator = hass.data[DOMAIN][entry.entry_id].pop("coordinator")
        if hasattr(coordinator, "shutdown"):
            coordinator.shutdown()
        
        # Pulisci completamente i dati del dominio per questa entry
        del hass.data[DOMAIN][entry.entry_id]

        # Pulisci la cache globale e le issue di riparazione per tutti gli add-on monitorati
        issue_registry: IssueRegistry = async_get_issue_registry(hass)
        for addon_slug in list(_DISCOVERED_ADDONS): # Iterare su una copia per permettere la modifica
            _DISCOVERED_ADDONS.remove(addon_slug)
            repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
            if issue_registry.get_issue(DOMAIN, repair_issue_id):
                delete_issue(hass, DOMAIN, repair_issue_id)
                _LOGGER.info("Eliminazione issue di riparazione '%s' durante lo scarico dell'integrazione.", repair_issue_id)
        
        _NOT_STARTED_TIMESTAMPS.clear() # Pulisci tutti i timestamp

    return unload_ok

