import { Box, Typography } from "@mui/material";
import { useEffect, useRef } from "react";
import type { CommandOutputLineSnapshot } from "../store/sratApi";

type TerminalChannel = "stdout" | "stderr" | "info";

type ReadonlyCommandTerminalProps = {
  lines?: CommandOutputLineSnapshot[] | null;
  maxHeight?: number;
  emptyText?: string;
};

const channelStyles: Record<string, { labelColor: string; textColor: string }> =
  {
    stdout: {
      labelColor: "text.secondary",
      textColor: "text.primary",
    },
    stderr: {
      labelColor: "error.main",
      textColor: "error.main",
    },
    info: {
      labelColor: "info.main",
      textColor: "info.main",
    },
  };

const channelPrefixPattern = /^\[(stdout|stderr|info)\]\s*/i;
const stderrLinePattern =
  /^(?:error:\s*|(?:check|format) failed\b|.*failed with exit code\b|.*errors? left uncorrected\b|.*exit status\b)/i;
const infoLinePattern =
  /^(?:starting\b|progress status not supported\b|filesystem (?:check|format)\b|(?:check|format) operation\b|(?:check|format) completed\b|(?:check|format) \/.+:)/i;

function normalizeTerminalLine(
  rawLine: string,
  fallbackChannel: TerminalChannel,
): { channel: TerminalChannel; line: string } | null {
  const trimmedLine = rawLine.trim();
  if (trimmedLine.length === 0) {
    return null;
  }

  const prefixedMatch = trimmedLine.match(channelPrefixPattern);
  if (prefixedMatch) {
    return {
      channel: prefixedMatch[1].toLowerCase() as TerminalChannel,
      line: trimmedLine.slice(prefixedMatch[0].length).trimStart(),
    };
  }

  if (/^ERROR:\s*/i.test(trimmedLine)) {
    return {
      channel: "stderr",
      line: trimmedLine.replace(/^ERROR:\s*/i, "").trimStart(),
    };
  }

  if (stderrLinePattern.test(trimmedLine)) {
    return { channel: "stderr", line: trimmedLine };
  }

  if (infoLinePattern.test(trimmedLine)) {
    return { channel: "info", line: trimmedLine };
  }

  return { channel: fallbackChannel, line: trimmedLine };
}

export function createTerminalLines(
  lines: string[],
  channel: TerminalChannel = "info",
  startTimestamp = Date.now(),
): CommandOutputLineSnapshot[] {
  return lines.flatMap((line, index) => {
    const normalizedLine = normalizeTerminalLine(line, channel);
    if (!normalizedLine) {
      return [];
    }

    return [
      {
        channel: normalizedLine.channel,
        line: normalizedLine.line,
        timestamp: startTimestamp + index,
      },
    ];
  });
}

export function ReadonlyCommandTerminal({
  lines,
  maxHeight = 420,
  emptyText = "No output available.",
}: ReadonlyCommandTerminalProps) {
  const outputLines = lines ?? [];
  const outputLinesLength = outputLines.length;
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (outputLinesLength > 0) {
      bottomRef.current?.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
      });
    }
  }, [outputLinesLength]);

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
      {outputLines.length === 0 ? (
        <Typography
          component="div"
          sx={{
            fontFamily: "inherit",
            fontSize: "inherit",
            color: "text.secondary",
          }}
        >
          {emptyText}
        </Typography>
      ) : (
        outputLines.map((line) => {
          const styles = channelStyles[line.channel] ?? channelStyles.stdout;
          return (
            <Typography
              key={`${line.timestamp}-${line.channel}-${line.line}`}
              component="div"
              sx={{
                fontFamily: "inherit",
                fontSize: "inherit",
                color: styles.textColor,
              }}
            >
              <Box
                component="span"
                sx={{
                  mr: 0.75,
                  fontWeight: 600,
                  color: styles.labelColor,
                }}
              >
                [{line.channel}]
              </Box>
              {line.line}
            </Typography>
          );
        })
      )}
      <div ref={bottomRef} />
    </Box>
  );
}
