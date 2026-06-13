import {
  ApplicationConfig,
  provideBrowserGlobalErrorListeners,
  importProvidersFrom
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { appRoutes } from './app.routes';
import { RocLendingApiModule, RocLendingConfiguration } from './roc-lending.module';


export const appConfig: ApplicationConfig = {
  providers: [provideBrowserGlobalErrorListeners(), provideRouter(appRoutes), importProvidersFrom(RocLendingApiModule.forRoot(() => new RocLendingConfiguration()))],
};
