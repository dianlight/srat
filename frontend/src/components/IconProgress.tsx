import HourglassEmptyIcon from "@mui/icons-material/HourglassEmpty";
import { Box, type SvgIconProps } from "@mui/material";
import type { SxProps, Theme } from "@mui/material/styles";
import { keyframes } from "@mui/material/styles";
import { type ElementType, useEffect, useMemo, useState } from "react";

const iconSpin = keyframes`
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
`;

const iconPulse = keyframes`
  0%,
  100% {
    opacity: 0.75;
    transform: scale(0.92);
  }
  50% {
    opacity: 1;
    transform: scale(1);
  }
`;

export interface IconProgressProps {
  icons: Array<ElementType<SvgIconProps>>;
  animationSpeed?: number;
  completeIcon?: ElementType<SvgIconProps>;
  completeIconColor?: SvgIconProps["color"];
  variant?: "determinate" | "indeterminate";
  value?: number;
  color?: SvgIconProps["color"];
  fontSize?: SvgIconProps["fontSize"];
  sx?: SxProps<Theme>;
}

export function IconProgress({
  icons,
  animationSpeed = 700,
  completeIcon,
  completeIconColor,
  variant,
  value,
  color = "inherit",
  fontSize = "small",
  sx,
}: IconProgressProps) {
  const [iconIndex, setIconIndex] = useState(0);

  const normalizedIcons = useMemo(
    () => (icons.length > 0 ? icons : [HourglassEmptyIcon]),
    [icons],
  );

  const resolvedVariant =
    variant ?? (typeof value === "number" ? "determinate" : "indeterminate");
  const isComplete =
    resolvedVariant === "determinate" &&
    typeof value === "number" &&
    value >= 100;
  const iconCount = normalizedIcons.length;
  const AnimatedIcon =
    normalizedIcons[iconIndex % iconCount] ?? HourglassEmptyIcon;
  const CurrentIcon = isComplete && completeIcon ? completeIcon : AnimatedIcon;

  useEffect(() => {
    if (isComplete || iconCount <= 1) {
      return;
    }

    const intervalId = setInterval(() => {
      setIconIndex((prev) => (prev + 1) % iconCount);
    }, animationSpeed);

    return () => {
      clearInterval(intervalId);
    };
  }, [animationSpeed, iconCount, isComplete]);

  return (
    <Box
      sx={[
        {
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Box
        sx={{
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          animation: isComplete
            ? undefined
            : `${iconPulse} 1.2s ease-in-out infinite`,
          color: "inherit",
          pointerEvents: "none",
        }}
      >
        <CurrentIcon
          color={isComplete && completeIconColor ? completeIconColor : color}
          fontSize={fontSize}
          sx={{
            animation: isComplete
              ? undefined
              : `${iconSpin} 1.4s linear infinite`,
          }}
        />
      </Box>
    </Box>
  );
}
