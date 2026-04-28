import {
  Body,
  Controller,
  Get,
  HttpCode,
  HttpStatus,
  Param,
  Post,
} from '@nestjs/common';
import {
  ApiBody,
  ApiOperation,
  ApiParam,
  ApiResponse,
  ApiTags,
} from '@nestjs/swagger';

import { MORTGAGE_EXAMPLE_APPLICATION_ID } from './constants';
import { CreditCheckDto } from './dto/credit-check.dto';
import { StartMortgageApplicationDto } from './dto/start-mortgage-application.dto';
import { MortgageService } from './mortgage.service';

@ApiTags('Applications')
@Controller('applications')
export class MortgageController {
  constructor(private readonly mortgageService: MortgageService) {}

  @Post()
  @HttpCode(HttpStatus.ACCEPTED)
  @ApiOperation({ summary: 'Start a mortgage application workflow' })
  @ApiBody({ type: StartMortgageApplicationDto })
  @ApiResponse({ status: 202, description: 'Workflow started' })
  @ApiResponse({ status: 409, description: 'Workflow already exists' })
  startApplication(@Body() dto: StartMortgageApplicationDto) {
    return this.mortgageService.startApplication(
      dto.applicationId,
      dto.applicantName,
      dto.scenario,
    );
  }

  @Get(':applicationId')
  @ApiOperation({ summary: 'Get mortgage application state' })
  @ApiParam({
    name: 'applicationId',
    type: String,
    example: MORTGAGE_EXAMPLE_APPLICATION_ID,
  })
  @ApiResponse({ status: 200, description: 'Current application state' })
  getApplication(@Param('applicationId') applicationId: string) {
    return this.mortgageService.getApplication(applicationId);
  }

  @Post(':applicationId/credit-check')
  @HttpCode(HttpStatus.ACCEPTED)
  @ApiOperation({ summary: 'Submit credit check result for an application' })
  @ApiParam({
    name: 'applicationId',
    type: String,
    example: MORTGAGE_EXAMPLE_APPLICATION_ID,
  })
  @ApiBody({ type: CreditCheckDto })
  @ApiResponse({ status: 202, description: 'Signal accepted' })
  completeCreditCheck(
    @Param('applicationId') applicationId: string,
    @Body() dto: CreditCheckDto,
  ) {
    return this.mortgageService.completeCreditCheck(
      applicationId,
      dto.result,
      dto.reference,
    );
  }
}
