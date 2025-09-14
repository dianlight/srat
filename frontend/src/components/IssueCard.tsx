import React from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Button,
  Box,
  Chip,
  IconButton,
  useTheme,
} from '@mui/material';
import {
  Error as ErrorIcon,
  Warning as WarningIcon,
  Info as InfoIcon,
  CheckCircle as SuccessIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { type Issue } from '../store/sratApi';
import { useIgnoredIssues } from '../hooks/issueHooks';

interface IssueCardProps {
  issue: Issue;
  onResolve?: (id: number) => void;
  showIgnored?: boolean;
}

const getSeverityConfig = (severity: string, theme: any) => {
  const isDark = theme.palette.mode === 'dark';

  switch (severity) {
    case 'error':
      return {
        color: theme.palette.error.main,
        backgroundColor: isDark
          ? theme.palette.error.dark + '20'
          : theme.palette.error.light + '40',
        icon: <ErrorIcon />,
        label: 'Error',
      };
    case 'warning':
      return {
        color: theme.palette.warning.main,
        backgroundColor: isDark
          ? theme.palette.warning.dark + '20'
          : theme.palette.warning.light + '40',
        icon: <WarningIcon />,
        label: 'Warning',
      };
    case 'info':
      return {
        color: theme.palette.info.main,
        backgroundColor: isDark
          ? theme.palette.info.dark + '20'
          : theme.palette.info.light + '40',
        icon: <InfoIcon />,
        label: 'Info',
      };
    case 'success':
      return {
        color: theme.palette.success.main,
        backgroundColor: isDark
          ? theme.palette.success.dark + '20'
          : theme.palette.success.light + '40',
        icon: <SuccessIcon />,
        label: 'Success',
      };
    default:
      return {
        color: theme.palette.text.secondary,
        backgroundColor: isDark ? theme.palette.grey[800] : theme.palette.grey[100],
        icon: <InfoIcon />,
        label: 'Unknown',
      };
  }
};

const IssueCard: React.FC<IssueCardProps> = ({ issue, onResolve, showIgnored }) => {
  const theme = useTheme();
  const { isIssueIgnored, ignoreIssue, unignoreIssue } = useIgnoredIssues();
  const isIgnored = isIssueIgnored(issue.id);
  const severityConfig = getSeverityConfig(issue.severity || 'info', theme);

  // When showIgnored is false, show only non-ignored items
  // When showIgnored is true, show all items
  if (!showIgnored && isIgnored) {
    return null;
  }

  return (
    <Card
      sx={{
        mb: 2,
        borderLeft: `4px solid ${severityConfig.color}`,
        backgroundColor: severityConfig.backgroundColor,
        opacity: issue.ignored ? 0.6 : 1,
      }}
    >
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <Box sx={{ color: severityConfig.color, mr: 1 }}>
            {severityConfig.icon}
          </Box>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            {issue.title}
          </Typography>
          <Chip
            label={severityConfig.label}
            size="small"
            sx={{
              backgroundColor: severityConfig.color,
              color: 'white',
              fontWeight: 'bold',
            }}
          />
          {issue.ignored && (
            <Chip
              label="Ignored"
              size="small"
              variant="outlined"
              sx={{ ml: 1, opacity: 0.7 }}
            />
          )}
        </Box>
        <Typography variant="body2" color="text.secondary">
          {issue.description}
        </Typography>
        {issue.date && (
          <Typography
            variant="caption"
            color="text.secondary"
            sx={{ mt: 1, display: 'block' }}
          >
            {new Date(issue.date).toLocaleString()}
          </Typography>
        )}
      </CardContent>
      <CardActions sx={{ justifyContent: 'space-between' }}>
        <Box>
          {!issue.ignored && onResolve && (
            <Button
              size="small"
              variant="outlined"
              sx={{ backgroundColor: severityConfig.color }}
              onClick={() => onResolve(issue.id)}
            >
              Resolve
            </Button>
          )}
        </Box>
        {onResolve && (
          <IconButton
            size="small"
            onClick={() => onResolve(issue.id)}
            title="Dismiss"
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        )}
      </CardActions>
    </Card>
  );
};

export default IssueCard;
