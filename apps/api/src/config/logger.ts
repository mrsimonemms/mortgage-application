import { ConsoleLoggerOptions, LogLevel } from '@nestjs/common';
import { registerAs } from '@nestjs/config';

// Allow for no logs to be emitted
export const LOG_LEVEL_SILENT = 'silent';

function generateLogLevels(): LogLevel[] {
  let currentLevel = process.env.LOG_LEVEL ?? 'log';
  if (currentLevel === 'info') {
    currentLevel = 'log';
  }
  let enabled = false;

  const levels: LogLevel[] = [
    'verbose',
    'debug',
    'log',
    'warn',
    'error',
    'fatal',
  ];

  if (
    !levels.includes(currentLevel as LogLevel) &&
    currentLevel !== LOG_LEVEL_SILENT
  ) {
    throw new Error(`Invalid log level provided: ${currentLevel}`);
  }

  // List the levels in order
  return levels.reduce((levels: LogLevel[], level: LogLevel) => {
    // Once a level is enabled, enable for everything higher than it
    if (level === currentLevel) {
      enabled = true;
    }

    // Once we reach the matching level, go for it
    if (enabled) {
      levels.push(level);
    }
    return levels;
  }, []);
}

export default registerAs(
  'logger',
  (): ConsoleLoggerOptions => ({
    logLevels: generateLogLevels(),
    colors: process.env.LOGGER_COLORS_ENABLED === 'true', // Default to false
    json: process.env.LOGGER_JSON_ENABLED !== 'false', // Default to true
  }),
);
