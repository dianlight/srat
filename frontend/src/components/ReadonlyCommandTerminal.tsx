import { Box, Typography } from "@mui/material";
import type { CommandOutputLineSnapshot } from "../store/sratApi";

type ReadonlyCommandTerminalProps = {
  lines?: CommandOutputLineSnapshot[] | null;
  maxHeight?: number;
  emptyText?: string;
};

export function ReadonlyCommandTerminal({
  lines,
  maxHeight = 420,
  emptyText = "No output available.",
}: ReadonlyCommandTerminalProps) {
  const outputLines = lines ?? [];

  return (
    <Box
      component="pre"
      sx={{
        m: 0,
        p: 1.5,
        maxHeight,
        overflow: "auto",
        bgcolor: "background.default",
        border: "1px solid",
        borderColor: "divider",
        borderRadius: 1,
        fontFamily: "monospace",
        fontSize: "0.78rem",
        whiteSpace: "pre-wrap",
      }}
    >
      {outputLines.length === 0
        ? emptyText
        : outputLines.map((line) => {
            const isStderr = line.channel === "stderr";
            return (
              <Typography
                key={`${line.timestamp}-${line.channel}-${line.line}`}
                component="div"
                sx={{
                  fontFamily: "inherit",
                  fontSize: "inherit",
                  color: isStderr ? "error.main" : "text.primary",
                }}
              >
                <Box
                  component="span"
                  sx={{
                    mr: 0.75,
                    fontWeight: 600,
                    color: isStderr ? "error.main" : "text.secondary",
                  }}
                >
                  [{line.channel}]
                </Box>
                {line.line}
              </Typography>
            );
          })}
    </Box>
  );
}
