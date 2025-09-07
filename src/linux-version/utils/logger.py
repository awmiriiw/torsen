import os
import logging
import coloredlogs
from config import setting
from logging.handlers import RotatingFileHandler


class RecordLog:
    """A centralized logging utility for the application."""
    
    def __init__(self, name: str):
        """Initializes and configures a logger instance."""
        
        self.logger = logging.getLogger(name)
        self.logger.setLevel(setting.GLOBAL_LOG_LEVEL)
        self.logger.propagate = False
        if self.logger.hasHandlers():
            return

        log_directory = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', setting.LOG_DIRECTORY)
        
        if not os.path.exists(log_directory):
            os.makedirs(log_directory)
            
        log_path = os.path.join(log_directory, setting.LOG_FILENAME)
        
        file_handler = RotatingFileHandler(log_path, maxBytes=setting.LOG_MAX_BYTES, backupCount=setting.LOG_BACKUP_COUNT, encoding='utf-8')
        file_handler.setLevel(setting.FILE_LOG_LEVEL)
        
        file_formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s (%(filename)s:%(lineno)d)')
        file_handler.setFormatter(file_formatter)
        
        coloredlogs.install(
            level=setting.CONSOLE_LOG_LEVEL,
            logger=self.logger,
            fmt='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
            level_styles={
                'debug': {'color': 'green'},
                'info': {'color': 'cyan'},
                'warning': {'color': 'yellow'},
                'error': {'color': 'red'},
                'critical': {'bold': True, 'color': 'red'},
            }
        )
        
        self.logger.addHandler(file_handler)

    def get_logger(self):
        """Returns the configured logger instance."""
        
        return self.logger