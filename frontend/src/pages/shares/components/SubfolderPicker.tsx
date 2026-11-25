import CreateNewFolderOutlinedIcon from "@mui/icons-material/CreateNewFolderOutlined";
import FolderOutlinedIcon from "@mui/icons-material/FolderOutlined";
import {
  Alert,
  Box,
  Breadcrumbs,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  InputAdornment,
  Link,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import {
  useGetApiSharesFoldersQuery,
  type VolumeFolders,
} from "../../../store/sratApi";
import { normalizeSubfolder, subfolderProblem } from "../utils";

interface SubfolderPickerProps {
  open: boolean;
  mountPath?: string;
  volumeLabel?: string;
  initialPath?: string;
  onSelect: (subfolder: string) => void;
  onClose: () => void;
  testOverrides?: {
    useGetApiSharesFoldersQuery?: typeof useGetApiSharesFoldersQuery;
  };
}

// SubfolderPicker browses the folders of a volume to pick a share subfolder
// (issue #184): click through folders, filter them by name, or name a new
// folder that is created when the share is saved.
export function SubfolderPicker({
  open,
  mountPath,
  volumeLabel,
  initialPath,
  onSelect,
  onClose,
  testOverrides,
}: SubfolderPickerProps) {
  const useFoldersQuery =
    testOverrides?.useGetApiSharesFoldersQuery ?? useGetApiSharesFoldersQuery;
  const [path, setPath] = useState("");
  // Segments typed as new folders do not exist yet: nothing to list.
  const [newFolders, setNewFolders] = useState(0);
  const [filter, setFilter] = useState("");
  const [newName, setNewName] = useState("");

  useEffect(() => {
    if (open) {
      setPath(normalizeSubfolder(initialPath));
      setNewFolders(0);
      setFilter("");
      setNewName("");
    }
  }, [open, initialPath]);

  const { data, isFetching, error } = useFoldersQuery(
    { mountPath: mountPath ?? "", path: path || undefined },
    { skip: !open || !mountPath || newFolders > 0 },
  );
  const listing = data as VolumeFolders | undefined;
  const segments = path === "" ? [] : path.split("/");

  const folders = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    return (listing?.folders ?? []).filter((name) =>
      name.toLowerCase().includes(needle),
    );
  }, [listing, filter]);

  const goTo = (next: string, createdSegments: number) => {
    setPath(next);
    setNewFolders(createdSegments);
    setFilter("");
  };
  const enter = (name: string) => goTo(path ? `${path}/${name}` : name, 0);
  const goUp = (depth: number) =>
    goTo(segments.slice(0, depth).join("/"), Math.max(0, newFolders - (segments.length - depth)));

  const newNameProblem = newName.trim() ? subfolderProblem(newName) : undefined;
  const addNewFolder = () => {
    const name = normalizeSubfolder(newName);
    if (!name || newNameProblem) return;
    goTo(path ? `${path}/${name}` : name, newFolders + name.split("/").length);
    setNewName("");
  };

  const errorMessage =
    error && "data" in error
      ? ((error.data as { detail?: string } | undefined)?.detail ?? "Cannot list this folder")
      : error
        ? "Cannot list this folder"
        : undefined;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm" aria-labelledby="subfolder-picker-title">
      <DialogTitle id="subfolder-picker-title">Choose a folder</DialogTitle>
      <DialogContent dividers>
        <Breadcrumbs aria-label="current folder" sx={{ mb: 2 }}>
          {[volumeLabel || mountPath || "Volume", ...segments].map((segment, depth) =>
            depth === segments.length ? (
              <Typography key={`/${segments.slice(0, depth).join("/")}`} color="text.primary">
                {segment}
              </Typography>
            ) : (
              <Link
                key={`/${segments.slice(0, depth).join("/")}`}
                component="button"
                type="button"
                underline="hover"
                onClick={() => goUp(depth)}
              >
                {segment}
              </Link>
            ),
          )}
        </Breadcrumbs>

        {newFolders > 0 ? (
          <Alert severity="info" sx={{ mb: 2 }}>
            This folder does not exist yet. It is created when you save the share.
          </Alert>
        ) : (
          <>
            <TextField
              size="small"
              fullWidth
              label="Filter folders"
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              sx={{ mb: 1 }}
            />
            {errorMessage ? (
              <Alert severity="error">{errorMessage}</Alert>
            ) : isFetching ? (
              <Box sx={{ display: "flex", justifyContent: "center", py: 3 }}>
                <CircularProgress size={28} aria-label="loading folders" />
              </Box>
            ) : folders.length === 0 ? (
              <Typography color="text.secondary" sx={{ py: 2 }}>
                {filter ? "No folder matches the filter." : "No folders here."}
              </Typography>
            ) : (
              <List dense sx={{ maxHeight: 320, overflow: "auto" }} aria-label="folders">
                {folders.map((name) => (
                  <ListItemButton key={name} onClick={() => enter(name)}>
                    <ListItemIcon>
                      <FolderOutlinedIcon />
                    </ListItemIcon>
                    <ListItemText primary={name} />
                  </ListItemButton>
                ))}
              </List>
            )}
            {listing?.truncated ? (
              <Typography variant="caption" color="text.secondary">
                Only the first {listing.folders.length} folders are listed. For a folder not in the list,
                type its path in the Subfolder field.
              </Typography>
            ) : null}
          </>
        )}

        <TextField
          size="small"
          fullWidth
          label="New folder"
          value={newName}
          onChange={(event) => setNewName(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              addNewFolder();
            }
          }}
          error={Boolean(newNameProblem)}
          helperText={newNameProblem ?? "Created inside the current folder when you save the share."}
          sx={{ mt: 2 }}
          slotProps={{
            input: {
              endAdornment: (
                <InputAdornment position="end">
                  <Tooltip title="Add new folder">
                    <span>
                      <IconButton
                        aria-label="add new folder"
                        disabled={!newName.trim() || Boolean(newNameProblem)}
                        onClick={addNewFolder}
                      >
                        <CreateNewFolderOutlinedIcon />
                      </IconButton>
                    </span>
                  </Tooltip>
                </InputAdornment>
              ),
            },
          }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={() => {
            onSelect(path);
            onClose();
          }}
        >
          {path ? `Use "${segments[segments.length - 1]}"` : "Use the whole volume"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
