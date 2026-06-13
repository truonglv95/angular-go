import { NgModule, ModuleWithProviders, SkipSelf, Optional, InjectionToken, Injectable } from '@angular/core';

// -----------------------------------------------------
// Mock classes representing external dependencies
// -----------------------------------------------------

import { DataModulesManager } from "@backbase/foundation-ang/data-http";
import { HttpClient } from "@angular/common/http";

export class RocLendingConfiguration {
  basePath: string = '/api/v1';
}

export const CONFIG_TOKEN = new InjectionToken<string>('CONFIG_TOKEN');

// -----------------------------------------------------
// The target module to test
// -----------------------------------------------------

@NgModule({
  imports: [],
  declarations: [],
  exports: [],
  providers: []
})
export class RocLendingApiModule {
  public static forRoot(configurationFactory: () => RocLendingConfiguration): ModuleWithProviders<RocLendingApiModule> {
    return {
      ngModule: RocLendingApiModule,
      providers: [ { provide: RocLendingConfiguration, useFactory: configurationFactory } ]
    };
  }

  constructor(
    @Optional() @SkipSelf() parentModule: RocLendingApiModule,
    @Optional() http: HttpClient,
    @Optional() dataModulesManager: DataModulesManager | undefined | null,
    config: RocLendingConfiguration,
  ) {
    if (parentModule) {
      throw new Error('RocLendingApiModule is already loaded. Import in your base AppModule only.');
    }
    if (!http) {
      console.warn('HttpClient is missing, but continuing for demo');
    }
    if (dataModulesManager) {
      dataModulesManager.setModuleConfig(CONFIG_TOKEN, {
        apiRoot: '',
        servicePath: config.basePath || '',
        headers: {},
      });
    }
  }
}
