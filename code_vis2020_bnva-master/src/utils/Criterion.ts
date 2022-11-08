export class ServiceCost {
  public serviceSpan = 10.0
  public timeInterval = 0.25
  public crewWage = 20.0
  public fuelCost = 1.5
  public maintenanceCost = 1

  // the time unit is hour
  public value (totalTime: number) {
    return this.serviceSpan * (2 * totalTime * this.crewWage + 2 * totalTime * (this.maintenanceCost + this.fuelCost) / 25.0) / this.timeInterval
  }
}

export class ConstructionCost {
  // unit: K
  public costPerStop = 10

  public value(stopNum: number) {
    return stopNum * this.costPerStop
  }
}
