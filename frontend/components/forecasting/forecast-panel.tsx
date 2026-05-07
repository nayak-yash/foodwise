import { ArrowDownRight, ChartNoAxesCombined, FlaskConical } from "lucide-react";
import type { DemandForecast } from "@/lib/types";

export function ForecastPanel({
  forecasts,
  showEvaluation,
}: {
  forecasts: DemandForecast[];
  showEvaluation: boolean;
}) {
  return (
    <section className="panel forecast-panel">
      <div className="panel-heading">
        <div>
          <div className="section-label">LOOKING AHEAD</div>
          <h2>
            Demand planning <span className="subtitle">{forecasts[0]?.weekday || "Today"}</span>
          </h2>
        </div>
        <span className="forecast-label">
          <ChartNoAxesCombined size={15} />
          Weekday-aware baseline
        </span>
      </div>
      {forecasts.length === 0 ? (
        <div className="empty-small">
          Forecasting service is starting. This panel refreshes automatically.
        </div>
      ) : (
        <div className="forecast-grid">
          {forecasts.map((forecast) => (
            <article className="forecast-card" key={forecast.ingredient_id}>
              <div className="forecast-card-heading">
                <h3>{forecast.ingredient}</h3>
                <ArrowDownRight size={17} />
              </div>
              <div className="forecast-number">
                {forecast.expected_kg}
                <span>kg expected today</span>
              </div>
              <div className="service-bars">
                {forecast.services.map((service) => (
                  <div key={service.service}>
                    <span>{service.service}</span>
                    <div>
                      <i
                        style={{
                          width: `${(service.expected_kg / Math.max(forecast.planning_upper_kg, 1)) * 100}%`,
                        }}
                      />
                    </div>
                    <strong>{service.expected_kg} kg</strong>
                  </div>
                ))}
              </div>
              <div className="forecast-bottom">
                <span>Suggested replenishment</span>
                <strong>{forecast.recommended_reorder_kg} kg</strong>
              </div>
              {showEvaluation && (
                <div className="backtest">
                  <strong>14-day holdout evaluation</strong>
                  <p>
                    Weekday model MAE: {forecast.backtest.weekday_mae_kg} kg
                    <br />
                    7-day mean baseline MAE: {forecast.backtest.seven_day_mean_mae_kg} kg
                  </p>
                  <small>{forecast.note}</small>
                </div>
              )}
            </article>
          ))}
        </div>
      )}
      <div className="panel-footer">
        <span>
          <FlaskConical size={13} />
          Forecasts use 84 days of seeded synthetic history.
        </span>
        <span>Lunch + dinner · UTC</span>
      </div>
    </section>
  );
}
