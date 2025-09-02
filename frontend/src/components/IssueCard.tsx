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
  onResolve: (id: number) => void;
  showIgnored?: boolean;
}

const getSeverityConfig = (severity: string) => {
  switch (severity) {
    case 'error':
      return {
        color: '#f44336',
        backgroundColor: '#ffebee',
        icon: <ErrorIcon />,
        label: 'Error',
      };
    case 'warning':
      return {
        color: '#ff9800',
        backgroundColor: '#fff3e0',
        icon: <WarningIcon />,
        label: 'Warning',
      };
    case 'info':
      return {
        color: '#2196f3',
        backgroundColor: '#e3f2fd',
        icon: <InfoIcon />,
        label: 'Info',
      };
    case 'success':
      return {
        color: '#4caf50',
        backgroundColor: '#e8f5e8',
        icon: <SuccessIcon />,
        label: 'Success',
      };
    default:
      return {
        color: '#757575',
        backgroundColor: '#f5f5f5',
        icon: <InfoIcon />,
        label: 'Unknown',
      };
  }
};

const IssueCard: React.FC<IssueCardProps> = ({ issue, onResolve, showIgnored }) => {
  const { isIssueIgnored, ignoreIssue, unignoreIssue } = useIgnoredIssues();
  const isIgnored = isIssueIgnored(issue.id);
  const severityConfig = getSeverityConfig(issue.severity || 'info');

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
          {!issue.ignored && (
            <Button
              size="small"
              variant="contained"
              sx={{ backgroundColor: severityConfig.color }}
              onClick={() => onResolve(issue.id)}
            >
              Resolve
            </Button>
          )}
        </Box>
        <IconButton
          size="small"
          onClick={() => onResolve(issue.id)}
          title="Dismiss"
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </CardActions>
    </Card>
  );
};

export default IssueCard;
